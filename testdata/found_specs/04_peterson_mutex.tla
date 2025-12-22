\* Peterson's Algorithm for Mutual Exclusion
\* Source: https://lamport.azurewebsites.net/tla/peterson.html
\* Classic two-process mutual exclusion algorithm using flags and turn variable

--algorithm Peterson {
   variables flag = [i \in {0, 1} |-> FALSE],  turn = 0;
   fair process (proc \in {0,1}) {
     a1: while (TRUE) {
          skip ;
     a2:  flag[self] := TRUE ;
     a3:  turn := 1 - self ;
     a4:  await (flag[1-self] = FALSE) \/ (turn = self);
     cs:  skip ;  \* Critical section
     a5:  flag[self] := FALSE
     }
   }
}
