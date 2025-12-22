\* Bounded Buffer Algorithm (Producer-Consumer)
\* Source: https://github.com/belaban/pluscal/blob/master/BoundedBuffer.tla
\* Models producers and consumers sharing a bounded buffer

--algorithm BoundedBuffer {
  variables
    buf = <<>>,
    accesses=0;

  process(p \in 1..PROD) {
    p0:
      while(accesses < 100) {
        either {
          p1:
            when Len(buf) < BUF_SIZE;
            accesses := accesses +1;
            buf := Append(buf, accesses);
          }
          or {
            p2: skip;
          }
        };
      }
  }

  process(c \in 5..CONS+5) {
    c0:
      while(accesses < 100) {
        either {
          c1:
            when Len(buf) > 0;
            buf := Tail(buf);
            accesses := accesses +1;
          }
          or {
            c2: skip;
          }
        };
      }
  }
}
