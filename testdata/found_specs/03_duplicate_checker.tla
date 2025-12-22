\* Duplication Checker Algorithm in PlusCal
\* Source: https://learntla.com/core/pluscal.html
\* Checks if a sequence contains duplicate elements using set operations

--algorithm dup
  variable seq = <<1, 2, 3, 2>>;
  index = 1;
  seen = {};
  is_unique = TRUE;

begin
  Iterate:
    while index <= Len(seq) do
      if seq[index] \notin seen then
        seen := seen \union {seq[index]};
      else
        is_unique := FALSE;
      end if;
      index := index + 1;
    end while;
end algorithm;
