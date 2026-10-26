{
  outputs =
    _:
    let
      pkgs = import (builtins.fetchTree {
        type = "github";
        owner = "NixOS";
        repo = "nixpkgs";
        rev = "55bfb4d7437bd4e09ab7cba58a43bd68eb310044";
        narHash = "sha256-jBYGPi4udQ9uo4KQp9m2J0546nJVQitG15gvZSaVVAs=";
      }) { system = "x86_64-linux"; };
      reviewHooks = builtins.fetchTree {
        type = "github";
        owner = "syntaqx";
        repo = "git-hooks";
        rev = "a3b888f92cd5b40b270c9a9752181fdc1717cbe5";
        narHash = "sha256-EGNMIGkfH5IAfAZ0ZvQ/lvCGiCEC3eRDKSdDU+qyfFg=";
      };
      reviewPreCommitConfig = pkgs.writeText "review-pre-commit.json" (
        builtins.toJSON {
          repos = [
            {
              repo = "local";
              hooks = [
                {
                  id = "go-fmt";
                  name = "go fmt";
                  language = "system";
                  entry = "${reviewHooks}/hooks/go-fmt.sh";
                  files = "\\.go$";
                }
                {
                  id = "trailing-whitespace";
                  name = "trim trailing whitespace";
                  language = "system";
                  entry = "${pkgs.python3Packages.pre-commit-hooks}/bin/trailing-whitespace-fixer";
                  types = [ "text" ];
                  stages = [
                    "pre-commit"
                    "pre-push"
                    "manual"
                  ];
                }
                {
                  id = "end-of-file-fixer";
                  name = "fix end of files";
                  language = "system";
                  entry = "${pkgs.python3Packages.pre-commit-hooks}/bin/end-of-file-fixer";
                  types = [ "text" ];
                  stages = [
                    "pre-commit"
                    "pre-push"
                    "manual"
                  ];
                }
                {
                  id = "check-yaml";
                  name = "check yaml";
                  language = "system";
                  entry = "${pkgs.python3Packages.pre-commit-hooks}/bin/check-yaml";
                  types = [ "yaml" ];
                }
                {
                  id = "check-added-large-files";
                  name = "check for added large files";
                  language = "system";
                  entry = "${pkgs.python3Packages.pre-commit-hooks}/bin/check-added-large-files";
                  stages = [
                    "pre-commit"
                    "pre-push"
                    "manual"
                  ];
                }
                {
                  id = "check-merge-conflict";
                  name = "check for merge conflicts";
                  language = "system";
                  entry = "${pkgs.python3Packages.pre-commit-hooks}/bin/check-merge-conflict";
                  types = [ "text" ];
                }
              ];
            }
          ];
        }
      );
      reviewEnvironment = ''
        if [ "''${REVIEW_SANDBOX:-}" != 1 ] || [ "$PWD" != /workspace ]; then exit 64; fi
        export TMPDIR=/workspace/.review/tmp GOCACHE=/workspace/.review/go-build
        export PRE_COMMIT_HOME=/workspace/.review/pre-commit
        export GOMODCACHE=/deps/go-modules GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local GOENV=off GOWORK=off GOFLAGS=-mod=readonly
        export GOMAXPROCS=4 AWS_EC2_METADATA_DISABLED=true
        mkdir -p "$TMPDIR" "$GOCACHE"
      '';
    in
    {
      devShells.x86_64-linux.review = pkgs.mkShell {
        shellHook = "";
        packages = [
          pkgs.gitMinimal
          pkgs.pre-commit
          pkgs.go
          pkgs.golangci-lint
          pkgs.python3Packages.pre-commit-hooks
        ];
        nativeBuildInputs = [
          (pkgs.writeShellApplication {
            name = "review-bootstrap";
            text = reviewEnvironment + "cd electronic-tariff-file-rotations; go mod verify";
          })
          (pkgs.writeShellApplication {
            name = "review-check";
            text = reviewEnvironment + ''
              status=0
              pre-commit run --all-files --config ${reviewPreCommitConfig} || status=$?
              cd electronic-tariff-file-rotations
              gofmt -d . | tee "$TMPDIR/format.diff" || status=$?
              if [ -s "$TMPDIR/format.diff" ]; then status=1; fi
              golangci-lint run ./... || status=$?
              go test -json ./... | tee "$TMPDIR/unit.jsonl" || status=$?
              CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$TMPDIR/bootstrap" . || status=$?
              echo 'The repository has no Go tests. The compiled handler is not executed: it can delete shared S3 objects.'
              exit "$status"
            '';
          })
        ];
        TZDIR = "${pkgs.tzdata.outPath}/share/zoneinfo";
        SSL_CERT_FILE = "${pkgs.cacert}/etc/ssl/certs/ca-bundle.crt";
      };
    };
}
