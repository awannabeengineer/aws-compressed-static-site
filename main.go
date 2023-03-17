package main

import (
	// "fmt"
	"io/fs"
	"mime"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/cloudfront"
	"github.com/pulumi/pulumi-aws/sdk/v5/go/aws/s3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func readFileOrPanic(path string) pulumi.StringInput {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err.Error())
	}
	return pulumi.String(data)
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Create an AWS resource (S3 Bucket)
		siteBucket, err := s3.NewBucket(ctx, "s3-site-bucket", &s3.BucketArgs{
			Website: s3.BucketWebsiteArgs{
				IndexDocument: pulumi.String("index.html"),
			},
		})
		if err != nil {
			return err
		}

		siteDir := "www" // Directory for content

		// create s3 object for each file
		// take care to address content_type for compressed files
		err = filepath.Walk(siteDir, func(name string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				rel, err := filepath.Rel(siteDir, name)
				if err != nil {
					return err
				}

				contentType := pulumi.String("")
				contentEncoding := pulumi.String("")

				if strings.HasSuffix(name, ".br") {
					contentType = pulumi.String(mime.TypeByExtension(path.Ext(strings.TrimSuffix(name, ".br"))))
					contentEncoding = "br"
				} else if strings.HasSuffix(name, ".gz") {
					contentType = pulumi.String(mime.TypeByExtension(path.Ext(strings.TrimSuffix(name, ".gz"))))
					contentEncoding = "gzip"

				} else {
					contentType = pulumi.String(mime.TypeByExtension(path.Ext(name)))
					contentEncoding = ""
				}

				if _, err := s3.NewBucketObject(ctx, rel, &s3.BucketObjectArgs{
					Bucket:          siteBucket.ID(),
					Source:          pulumi.NewFileAsset(name),
					ContentType:     contentType,
					ContentEncoding: contentEncoding,
				}); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}

		s3OriginId := "mySiteOrigin"

		// Set access policy so all objects are readable
		if _, err := s3.NewBucketPolicy(ctx, "bucketPolicy", &s3.BucketPolicyArgs{
			Bucket: siteBucket.ID(),
			Policy: pulumi.Any(map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []map[string]interface{}{
					{
						"Effect":    "Allow",
						"Principal": "*",
						"Action": []interface{}{
							"s3:GetObject",
						},
						"Resource": []interface{}{
							pulumi.Sprintf("arn:aws:s3:::%s/*", siteBucket.ID()),
						},
					},
				},
			}),
		}); err != nil {
			return err
		}

		// create and associate cloudfront function with endpoint
		cFunction, err := cloudfront.NewFunction(ctx, "redirect_compressed_and_cache", &cloudfront.FunctionArgs{
			Runtime: pulumi.String("cloudfront-js-1.0"),
			Comment: pulumi.String("find cause of problem with precompressed selector logic"),
			Publish: pulumi.Bool(true),
			Code:    readFileOrPanic("./function.js"),
		})
		if err != nil {
			return err
		}

		// create a cache policy for cloudfront distribution
    cPolicy, err := cloudfront.NewCachePolicy(ctx, "cachePolicy", &cloudfront.CachePolicyArgs{
			Comment:    pulumi.String("Cache policy for static site"),
			Name:       pulumi.String("precompressed_static_site_cache_policy"),
			DefaultTtl: pulumi.Int(3600),
			MaxTtl:     pulumi.Int(86400),
			MinTtl:     pulumi.Int(0),
			ParametersInCacheKeyAndForwardedToOrigin: cloudfront.CachePolicyParametersInCacheKeyAndForwardedToOriginArgs{
				CookiesConfig: &cloudfront.CachePolicyParametersInCacheKeyAndForwardedToOriginCookiesConfigArgs{
					CookieBehavior: pulumi.String("none"),
				},
				EnableAcceptEncodingBrotli: pulumi.Bool(false),
				EnableAcceptEncodingGzip:   pulumi.Bool(false),
				HeadersConfig: &cloudfront.CachePolicyParametersInCacheKeyAndForwardedToOriginHeadersConfigArgs{
					HeaderBehavior: pulumi.String("whitelist"),
					Headers: &cloudfront.CachePolicyParametersInCacheKeyAndForwardedToOriginHeadersConfigHeadersArgs{
						Items: pulumi.StringArray{
							pulumi.String("Accept-Encoding"),
						},
					},
				},
				QueryStringsConfig: &cloudfront.CachePolicyParametersInCacheKeyAndForwardedToOriginQueryStringsConfigArgs{
					QueryStringBehavior: pulumi.String("none"),
				},
			},
		})
		if err != nil {
			return err
		}

		// create a cloudfront distribution and set this bucket as origin
		_, err = cloudfront.NewDistribution(ctx, "s3Distribution", &cloudfront.DistributionArgs{
			Origins: cloudfront.DistributionOriginArray{
				&cloudfront.DistributionOriginArgs{
					DomainName: siteBucket.BucketDomainName,
					// OriginAccessControlId: pulumi.Any(),
					OriginId: pulumi.String(s3OriginId),
				},
			},
			Enabled:           pulumi.Bool(true),
			IsIpv6Enabled:     pulumi.Bool(true),
			Comment:           pulumi.String("Static Site distribution"),
			DefaultRootObject: pulumi.String("index.html"),
			LoggingConfig: &cloudfront.DistributionLoggingConfigArgs{
				IncludeCookies: pulumi.Bool(false),
				Bucket:         pulumi.String("wbe-logs.s3.amazonaws.com"),
				Prefix:         pulumi.String("site"),
			},
			// Aliases: pulumi.StringArray{
			// 	pulumi.String("staticaws.arashfarr.com"),
			// },

			DefaultCacheBehavior: &cloudfront.DistributionDefaultCacheBehaviorArgs{
				AllowedMethods: pulumi.StringArray{
					pulumi.String("GET"),
					pulumi.String("HEAD"),
				},
				CachedMethods: pulumi.StringArray{
					pulumi.String("GET"),
					pulumi.String("HEAD"),
				},

				// create function association for cloudfront distribution
				FunctionAssociations: cloudfront.DistributionDefaultCacheBehaviorFunctionAssociationArray{
					&cloudfront.DistributionDefaultCacheBehaviorFunctionAssociationArgs{
						// function arn
						FunctionArn: cFunction.Arn,
						// Event type
						EventType: pulumi.String("viewer-request"),
					},
				},

				// add a cache policy to the distribution
				CachePolicyId: cPolicy.ID(),

				TargetOriginId:       pulumi.String(s3OriginId),
				ViewerProtocolPolicy: pulumi.String("allow-all"),
				MinTtl:               pulumi.Int(0),
				DefaultTtl:           pulumi.Int(3600),
				MaxTtl:               pulumi.Int(86400),
			},
			PriceClass: pulumi.String("PriceClass_200"),
			Restrictions: &cloudfront.DistributionRestrictionsArgs{
				GeoRestriction: &cloudfront.DistributionRestrictionsGeoRestrictionArgs{
					RestrictionType: pulumi.String("whitelist"),
					Locations: pulumi.StringArray{
						pulumi.String("US"),
						pulumi.String("CA"),
						pulumi.String("GB"),
						pulumi.String("DE"),
					},
				},
			},
			Tags: pulumi.StringMap{
				"Environment": pulumi.String("production"),
			},

			ViewerCertificate: &cloudfront.DistributionViewerCertificateArgs{
				CloudfrontDefaultCertificate: pulumi.Bool(true),
			},
		})
		if err != nil {
			return err
		}

		// Export the name of the bucket
		ctx.Export("bucketName", siteBucket.ID())
		ctx.Export("websiteUrl", siteBucket.WebsiteEndpoint)
		return nil
	})
}
