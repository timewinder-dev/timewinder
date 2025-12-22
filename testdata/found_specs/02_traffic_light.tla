\* Traffic Light Controller in PlusCal
\* Source: https://writinglinesofcode.com/blog/pluscal-part-01
\* Models a simple traffic light that cycles through red, yellow, and green states

(* --algorithm PlusCalTrafficLight
{
  variables red = 1, yellow = 0, green = 0;
  {
    TurnGreen: if (red = 1 /\ yellow = 0 /\ green = 0)
               { red := 0; green := 1; };
    TurnYellow: if (red = 0 /\ yellow = 0 /\ green = 1)
                { green := 0; yellow := 1; };
    TurnRed: if (red = 0 /\ yellow = 1 /\ green = 0)
             { red := 1; yellow := 0; };
  }
}
*)
