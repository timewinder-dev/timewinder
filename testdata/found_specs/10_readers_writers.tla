\* Readers-Writers Problem
\* Classic synchronization problem where multiple readers can read simultaneously
\* but writers need exclusive access
\* Source: Classic OS/concurrency textbook problem

--algorithm ReadersWriters {
  variables
    readers = 0,
    writing = FALSE,
    data = 0;

  process (reader \in 1..2) {
    r1:
      await ~writing;
      readers := readers + 1;
    r2:
      skip;  \* Read data
    r3:
      readers := readers - 1;
  }

  process (writer \in 3..4) {
    w1:
      await ~writing /\ readers = 0;
      writing := TRUE;
    w2:
      data := data + 1;  \* Write data
    w3:
      writing := FALSE;
  }
}
