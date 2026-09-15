.PHONY: dev build test clean

dev:
	wails dev

build:
	wails generate module
	(cd frontend && sh build.sh)
	wails build
	xattr -cr "build/bin/Termius Tools.app" 2>/dev/null || xattr -cr build/bin/termius-tools.app 2>/dev/null || true

test:
	go test ./...

clean:
	rm -rf build/bin
