let count = 0;

function incrementCounter() {
  count++;
  document.getElementById("counter").textContent = count;
}

function decrementCounter() {
  count--;
  document.getElementById("counter").textContent = count;
}
