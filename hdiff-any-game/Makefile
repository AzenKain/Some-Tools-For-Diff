APP_NAME = hdiff-any-game.exe

all: build

build:
	go build -o $(APP_NAME) ./

run: build
	./$(APP_NAME)

clean:
	rm -f $(APP_NAME) diff.txt