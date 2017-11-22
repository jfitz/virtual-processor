echo
TESTROOT=$1
TESTBED=$2
TESTGROUP=$3
TESTNAME=$4
OPTIONS=$5
echo Start test $TESTNAME

# create testbed
echo Creating testbed...
mkdir "$TESTBED/$TESTNAME"
cp "$TESTROOT/$TESTGROUP/$TESTNAME/data"/* "$TESTBED/$TESTNAME"
echo testbed ready

# execute program
ECODE=0

echo Running program...
go run assembler/assembler.go "$TESTBED/$TESTNAME/program.asm" "$TESTBED/$TESTNAME/program.module" >"$TESTBED/$TESTNAME/stdout.txt" 2>&1
xxd -g 1 "$TESTBED/$TESTNAME/program.module" >"$TESTBED/$TESTNAME/module.dump"
echo run finished

# compare results
echo Comparing stdout...
diff "$TESTBED/$TESTNAME/stdout.txt" "$TESTROOT/$TESTGROUP/$TESTNAME/ref/program.list"
((ECODE+=$?))
echo compare done

echo Comparing module...
diff "$TESTBED/$TESTNAME/module.dump" "$TESTROOT/$TESTGROUP/$TESTNAME/ref/module.dump"
((ECODE+=$?))
echo compare done

echo End test $TESTNAME
exit $ECODE
