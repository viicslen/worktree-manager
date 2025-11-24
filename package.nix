{ lib
, buildGoModule
, git
}:

buildGoModule rec {
  pname = "wtm";
  version = "0.1.0";

  src = ./.;

  vendorHash = "sha256-pprnK2JKmPuR3Q+F8+vMDEdowlb3oX4BOOzW8NGOqgs=";

  ldflags = [
    "-s"
    "-w"
    "-X main.version=${version}"
  ];

  # Skip tests as they require git repository setup
  doCheck = false;

  # Ensure git is available at runtime
  nativeBuildInputs = [ git ];
  propagatedBuildInputs = [ git ];

  meta = with lib; {
    description = "Worktree Manager - A CLI tool for managing git worktrees";
    homepage = "https://github.com/viicslen/worktrees";
    license = licenses.mit;
    maintainers = [ ];
    mainProgram = "wtm";
  };
}
