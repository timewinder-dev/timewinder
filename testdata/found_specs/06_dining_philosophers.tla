\* Dining Philosophers Problem
\* Source: https://github.com/muratdem/PlusCal-examples/blob/master/DiningPhil/diningRound0.tla
\* Classic concurrency problem modeling philosophers sharing forks around a table

--algorithm diningRound0 {
  variable fork = [k \in Procs |-> N];

  define {
    forkAvailable(i) == fork[i]=N
    LeftF(i) == i
    LeftP(i) == IF (i=N-1) THEN 0 ELSE i+1
    RightF(i) == IF (i=0) THEN (N-1) ELSE (i-1)
    RightP(i) == IF (i=0) THEN (N-1) ELSE (i-1)
  }

  fair process (j \in Procs)
  variable state = "Thinking";
  {
    J0: while (TRUE) {
      H: either {
        if (state="Thinking") state:="Hungry";
      }
      or P: {
        if (state="Hungry") {
          await(forkAvailable(RightF(self)));
          fork[RightF(self)]:=self;
          E: await(forkAvailable(LeftF(self)));
          fork[LeftF(self)]:=self;
          state:="Eating";
        }
      }
      or T: {
        if (state="Eating") {
          state:="Thinking";
          fork[LeftF(self)]:=N;
          R: fork[RightF(self)]:=N;
        }
      }
    }
  }
}
