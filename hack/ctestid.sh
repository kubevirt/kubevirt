oldnum=$(cut -d ',' -f2 ctestid.txt)
newnum=$(expr $oldnum + 1)
sed -i "s/$oldnum\$/$newnum/g" ctestid.txt

echo $newnum
