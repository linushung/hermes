1. Install the Golang tools: [***Installer***](https://golang.org/dl/)

2. Check that Go is installed correctly
    - Go distribution in **`/usr/local/go`**
    - **`/usr/local/go/bin`** directory in your PATH

3. Set up a workspace and .bash_profile | .profile for GOPATH
    - *export GOPATH=$HOME/go*
    - *export GOBIN=$GOPATH/bin*
    - *export PATH=$PATH:$GOBIN*

4. Check IDE or code editor's go-plugin uses **goimports** for formatting code

5. Install missing dependency packages of hermes
   ```
   make clean
   ```

6. Start project
   ```
   make hermes
   ```
