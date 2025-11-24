{ lib
, buildGoModule
, git
}:

buildGoModule rec {
  pname = "wtm";
  version = "0.1.3";

  src = ./.;

  vendorHash = "sha256-pprnK2JKmPuR3Q+F8+vMDEdowlb3oX4BOOzW8NGOqgs=";

  ldflags = [
    "-s"
    "-w"
    "-X main.version=${version}"
  ];

  # Ensure git is available for tests and runtime
  nativeBuildInputs = [ git ];
  propagatedBuildInputs = [ git ];

  # Configure git for tests
  preCheck = ''
    export HOME=$TMPDIR
    git config --global user.email "test@example.com"
    git config --global user.name "Test User"
    git config --global init.defaultBranch main
  '';

  meta = with lib; {
    description = "Worktree Manager - A CLI tool for managing git worktrees";
    homepage = "https://github.com/viicslen/worktrees";
    license = licenses.mit;
    maintainers = [ ];
    mainProgram = "wtm";
  };
}
