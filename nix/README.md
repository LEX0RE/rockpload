## Nix

To test the Nix workflow for development, you can use your own system or a Docker container: 

```bash
docker run -it --rm nixos/nix /bin/sh
```

## 1. Enable modern Nix features (Flakes)

Inside the container, you first need to enable `nix-command` and `flakes` to use the modern Nix setup:

```bash
mkdir -p ~/.config/nix
echo "experimental-features = nix-command flakes" >> ~/.config/nix/nix.conf
```

## 2. Clone the project

Clone the repository to get the source code:

```bash
git clone https://github.com/LEX0RE/rockpload.git
cd rockpload
```

## 3. Build the project with Nix

Verify that the project builds successfully. This will automatically download the necessary dependencies and compile the application:

```bash
nix build .
```
*(Note: If successful, a `result` symlink containing the compiled binary will be created in your current directory).*

## 4. Enter the Nix development environment (optional)

To test the reproducible workspace, enter the development shell. This downloads all the Go/C dependencies required to work on the project:

```bash
nix develop
```
