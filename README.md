Virtual Processor

Inspired by CBASIC, the UCSD p-System, the JVM, and .NET, I have built my own virtual processor and assembler.

My intention is to compile BASIC programs (see my BASIC-1965 and BASIC-1973 projects) and run them without the interpreter. I'm cheating on the compiler; my plan is to have the interpreter, after reading and parsing the code, emit assembly language.

This virtual processor is stack-oriented with no registers. It uses an architecture that varies from the typical Von-Neumann; it has multiple memory segments: code, data, value stack, and return stack. Code can modify data but not itself, and values are pushed onto one stack separate from return values which are pushed onto a different stack.

I have no interest in efficiency, so programs may run slowly. (I don't know at this point.)

