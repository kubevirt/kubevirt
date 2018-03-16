#!/bin/bash
while ((1))
do
SM=( "SLAVE" "SLAVE" "SLAVE")
idx=$((RANDOM % 3))
SM[$idx]="MASTER"
cat <<~~
-=# SuperCLI #=-
Please choose one controller from the list:
  0 -- Controller1  (${SM[0]})
  1 -- Controller2  (${SM[1]})
  2 -- Controller3  (${SM[2]})
Please enter a choice (by default, connect to master):
~~
  # c: '0 -- .*\(MASTER\)' 0\n
  # c: '1 -- .*\(MASTER\)' 1\n
  # c: '2 -- .*\(MASTER\)' 2\n
  read i
  if [[ $i != $idx ]]
  then
   exit 1
  fi
  # e: Controller[0-2]/>
  # s: \n
  echo -n "Controller$idx/> "
  read
done
