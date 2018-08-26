TESTROOT=test
SRCGROUP=assembler
DESTGROUP=runner

echo Migrating all tests...
ECODE=0

for F in "$TESTROOT/$SRCGROUP"/*; do
    if [ -e "$TESTROOT/$SRCGROUP/${F##*/}/ref/program.module" ]
    then
	cp "$TESTROOT/$SRCGROUP/${F##*/}/ref/program.module" "$TESTROOT/$DESTGROUP/${F##*/}/data"
    fi
done
