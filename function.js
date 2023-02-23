function handler(event) {
  var request = event.request;
  var headers = request.headers;
  if (request.uri.endsWith('/')) {
    request.uri += 'index.html';
  }
  var uri = request.uri;

  // Log request data and redirect URL
  var output = {};
  output[request.method] = uri;
  output.Headers = headers;
  console.log(output);

  // redirect uri request based of accept-encoding encoding headers
  if (headers['accept-encoding'] && headers['accept-encoding'].value.includes('br')) {
    request.uri = uri + '.br';
  }
  else if (headers['accept-encoding'] && headers['accept-encoding'].value.includes('gzip')) {
    request.uri = uri + '.gz';
  }
  return request;
}
