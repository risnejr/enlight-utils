# Enlight utilities 

## `gen_config`
Generates a `config.json` file for services that need access to enlight hierarchy structures. To generate a binary for your architecture simply run `$ go build`. Then run the binary and make sure that the certificate folder is in the same directory as your binary.

## `build_hierarchy` 
Generates an **Enlight hierarchy** based on the **RPi's** hostname. The generated inspection points are based on the data which is collectable from the **BME680** sensor. Thus temperature, humidity, pressure and volatile gases. 