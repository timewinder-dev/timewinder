\* Concurrent Counter
\* Simple example showing race conditions with concurrent increments
\* Common teaching example in concurrency courses

--algorithm ConcurrentCounter {
  variables counter = 0;

  process (inc \in 1..2) {
    loop:
      while (counter < 10) {
        read:
          with (temp = counter) {
        write:
            counter := temp + 1;
          }
      }
  }
}
