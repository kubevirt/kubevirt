#!/bin/sh
tempFile=$(mktemp --dry-run)
echo $4 > $tempFile
sed -i "s|<baseBoard></baseBoard>|<baseBoard><entry name='manufacturer'>Radical Edward</entry></baseBoard>|" $tempFile
cat $tempFile