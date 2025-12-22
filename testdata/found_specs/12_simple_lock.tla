\* Simple Lock/Mutex Implementation
\* Basic mutual exclusion using a single boolean lock variable
\* Demonstrates the simplest form of synchronization
\* Source: Fundamental concurrency pattern

--algorithm SimpleLock {
  variables
    lock = FALSE,
    critical_count = 0;

  process (proc \in 1..2) {
    loop:
      while (critical_count < 3) {
        acquire:
          await ~lock;
          lock := TRUE;
        critical:
          critical_count := critical_count + 1;
        release:
          lock := FALSE;
      }
  }
}
