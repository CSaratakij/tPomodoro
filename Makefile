APPNAME = tPomodoro
VERSION = 1.1.0

all: clean build dist

run:
	./build/$(APPNAME)

build:
	go mod download
	go build main.go
	mkdir build
	mv main build/$(APPNAME)

dist:
	cp LICENSE build/LICENSE
	cp tPomodoro-alert.sh build/tPomodoro-alert.sh
	cp -a res/ build/res
	zip -r $(APPNAME)_$(VERSION).zip build
	mv $(APPNAME)_$(VERSION).zip build/

clean:
	rm -rf build/
