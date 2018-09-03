TESTROOT=test
SRCGROUP=assembler
DESTGROUP=runner

echo Migrating all tests...
ECODE=0

for F in "$TESTROOT/$SRCGROUP"/*; do
    FILENAME=${F##*/}
    if [ -e "$TESTROOT/$SRCGROUP/$FILENAME/ref/program.module" ] && [ -e "$TESTROOT/$DESTGROUP/$FILENAME/data/program.module" ]
    then
	echo Copying "$TESTROOT/$SRCGROUP/$FILENAME/ref/program.module" to "$TESTROOT/$DESTGROUP/$FILENAME/data"
	cp "$TESTROOT/$SRCGROUP/$FILENAME/ref/program.module" "$TESTROOT/$DESTGROUP/$FILENAME/data"
    fi
done
