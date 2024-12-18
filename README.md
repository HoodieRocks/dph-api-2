# Datapack Hub API (Go Rewrite)

A Go port of an older group project, with some upgrades, and personal changes. The original API was written in Python
and was filled with bugs, and the code was horrendous. This rewrite uses Go and will use better practices and
technologies.

## Hosting

This project is being made for fun and to learn some of the ins-and-outs of Go, and therefore will not be hosted, if you
wish to host this for whatever reason, here's the steps you need:

1. Configure Postgres
    - Install Postgres 16
    - Create database
    - Export login url as an environment variable
2. Install Go (use your preferred package manager)
3. Run the script
    - run `go build -o dist/server.exe`
    - run the generated executable
4. Assuming I didn't mess up tables, you should now have a fully running high-performance, bare-bones rewrite of the DPH
   API

## Acknowledgements of AI

The AI used to generate some of the repetitive error statements and commenting was [Codeium](https://codeium.com/).
Other than that, most if not all of the functionality code was written by me.

## Contributing

Contributions would be greatly appreciated. We recommend forking the project and making pull requests if you choose to
contribute code. If you need to report issues, please use the issues tab on GitHub
