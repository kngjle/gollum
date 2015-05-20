.PHONY: all clean freebsd linux mac pi win
clean:
	@rm -f ./gollum
	@rm -f ./gollum_*.tar.gz

linux:
	@echo "Building for Linux"
	@GOOS=linux GOARCH=amd64 go build -o gollum
	@tar czf gollum_linux.tar.gz gollum gollum_*.conf

mac:
	@echo "Building for MacOS X"
	@GOOS=darwin GOARCH=amd64 go build -o gollum
	@tar czf gollum_mac.tar.gz gollum gollum_*.conf

freebsd:
	@echo "Building for FreeBSD"
	@GOOS=freebsd GOARCH=amd64 go build -o gollum
	@tar czf gollum_freebsd.tar.gz gollum gollum_*.conf

win:
	@echo "Building for Windows"
	@GOOS=windows GOARCH=amd64 go build -o gollum
	@tar czf gollum_win.tar.gz gollum gollum_*.conf

pi:
	@echo "Building for Raspberry Pi"
	@GOOS=linux GOARCH=arm go build -o gollum
	@tar czf gollum_pi.tar.gz gollum gollum_*.conf

all: clean freebsd linux mac pi win