# Configuration Parser

A library which can be used to pull configuration values from either environment variables or command line flags.

You pass a struct to configparser, and it populates that struct with values pulled from environment variables and command line flags. You can also tell configparser how you want various fields in the struct to be parsed by using tags in the struct.