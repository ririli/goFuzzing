## Usage
1. `GoPie` has been implemented using `Go 1.19.1`. 

    Follow https://go.dev/doc/install to install the right version of Go.
2. Build the binary under `cmd` with `go build -o ./bin ./cmd/...`. There will be two binaries after compilation, the `inst` and `fuzz`. 
    ~~~shell
    go build -o ./bin ./cmd/...
    ~~~
2. Instrument stubs and compile your project which is to be tested.
    ~~~shell
    ./bin/fuzz --task inst --path your_project_to_be_tested
    ~~~
3. Build test binaries, the test binaries will be placed into `./testbins`
    // compile the unit tests,use -o
3. ~~~shell
    ./bin/fuzz --task bins --path your_project_to_be_tested
    ~~~
4. Start testing
    ~~~shell
   //use > xx.txt 
    ./bin/fuzz --task full --path path_of_test_binaries
    ~~~
