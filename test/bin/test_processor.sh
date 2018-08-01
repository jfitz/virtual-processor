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
go run processor/processor.go --trace "$TESTBED/$TESTNAME/program.module" >"$TESTBED/$TESTNAME/stdout.txt" 2>&1
echo run finished

# compare results
echo Comparing stdout...
diff "$TESTBED/$TESTNAME/stdout.txt" "$TESTROOT/$TESTGROUP/$TESTNAME/ref/stdout.txt"
((ECODE+=$?))

if [ $ECODE -ne 0 ]
then
    cp "$TESTBED/$TESTNAME/stdout.txt" "$TESTROOT/$TESTGROUP/$TESTNAME/ref/stdout.txt"
fi

echo compare done

echo End test $TESTNAME
exit $ECODE
