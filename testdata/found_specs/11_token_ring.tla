\* Token Ring Protocol
\* Models a simple token passing protocol for mutual exclusion
\* Only the process holding the token can enter its critical section
\* Source: Classic distributed algorithm pattern

--algorithm TokenRing {
  variables
    token = 0,  \* Process ID that currently has the token
    in_cs = [i \in 0..2 |-> FALSE];

  process (proc \in 0..2) {
    loop:
      while (TRUE) {
        wait_token:
          await token = self;
        enter_cs:
          in_cs[self] := TRUE;
        exit_cs:
          in_cs[self] := FALSE;
        pass_token:
          token := (self + 1) % 3;
      }
  }
}
