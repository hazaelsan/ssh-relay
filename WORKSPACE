workspace(name = "ssh_relay")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

# Bazel core.
http_archive(
    name = "bazel_skylib",
    sha256 = "97e70364e9249702246c0e9444bccdc4b847bed1eb03c5a3ece4f83dfe6abc44",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-skylib/releases/download/1.0.2/bazel-skylib-1.0.2.tar.gz",
        "https://github.com/bazelbuild/bazel-skylib/releases/download/1.0.2/bazel-skylib-1.0.2.tar.gz",
    ],
)

load("@bazel_skylib//:workspace.bzl", "bazel_skylib_workspace")

bazel_skylib_workspace()

http_archive(
    name = "rules_proto",
    sha256 = "602e7161d9195e50246177e7c55b2f39950a9cf7366f74ed5f22fd45750cd208",
    strip_prefix = "rules_proto-97d8af4dc474595af3900dd85cb3a29ad28cc313",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_proto/archive/97d8af4dc474595af3900dd85cb3a29ad28cc313.tar.gz",
        "https://github.com/bazelbuild/rules_proto/archive/97d8af4dc474595af3900dd85cb3a29ad28cc313.tar.gz",
    ],
)

load("@rules_proto//proto:repositories.bzl", "rules_proto_dependencies", "rules_proto_toolchains")

rules_proto_dependencies()

rules_proto_toolchains()

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "a8d6b1b354d371a646d2f7927319974e0f9e52f73a2452d2b3877118169eb6bb",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.23.3/rules_go-v0.23.3.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.23.3/rules_go-v0.23.3.tar.gz",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "cdb02a887a7187ea4d5a27452311a75ed8637379a1287d8eeb952138ea485f7d",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.21.1/bazel-gazelle-v0.21.1.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.21.1/bazel-gazelle-v0.21.1.tar.gz",
    ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

gazelle_dependencies()

# Deps.
http_archive(
    name = "com_github_googleapis_googleapis",
    sha256 = "bb28f5d28f095a013dca653be097c51423d395da62a00d58c55cfe4efb92951b",
    strip_prefix = "googleapis-3f5f8a2258c6a41f9fbf7b80acbca631dda0a952",
    url = "https://github.com/googleapis/googleapis/archive/3f5f8a2258c6a41f9fbf7b80acbca631dda0a952.zip",
)

load("@com_github_googleapis_googleapis//:repository_rules.bzl", "switched_rules_by_language")

switched_rules_by_language(
    name = "com_google_googleapis_imports",
    go = True,
    grpc = True,
)

http_archive(
    name = "com_github_golang_glog",
    build_file = "bazel/BUILD.golang_glog",
    sha256 = "433e5b9696e71828a109cef978312d650dd78f51eb3fd4dc59656013932dd0d1",
    strip_prefix = "glog-23def4e6c14b4da8ac2ed8007337bc5eb5007998",
    url = "https://github.com/golang/glog/archive/23def4e6c14b4da8ac2ed8007337bc5eb5007998.zip",
)

http_archive(
    name = "com_github_google_uuid",
    build_file = "bazel/BUILD.google_uuid",
    sha256 = "b2041c5847c227bb5feee5fa5aaa32736e094a2d26357e66fcc5cc405c36386a",
    strip_prefix = "uuid-1.1.1",
    urls = ["https://github.com/google/uuid/archive/v1.1.1.zip"],
)

http_archive(
    name = "com_github_gorilla_websocket",
    build_file = "bazel/BUILD.gorilla_websocket",
    sha256 = "335a84a456112cce890d00ec82e59cfe5a07581f53bd0b9284d21714d0527bc1",
    strip_prefix = "websocket-1.4.2",
    urls = ["https://github.com/gorilla/websocket/archive/v1.4.2.zip"],
)

http_archive(
    name = "com_github_kylelemons_godebug",
    build_file = "bazel/BUILD.kylelemons_godebug",
    sha256 = "a07edfa7b01c277196479e1ec51b92b416f2935c049f96917632e9c000e146f8",
    strip_prefix = "godebug-1.1.0",
    url = "https://github.com/kylelemons/godebug/archive/v1.1.0.zip",
)

go_repository(
    name = "org_golang_google_grpc",
    importpath = "google.golang.org/grpc",
    tag = "v1.29.1",
)

go_repository(
    name = "org_golang_x_net",
    commit = "627f964",
    importpath = "golang.org/x/net",
)

go_repository(
    name = "org_golang_x_sys",
    commit = "f1bc736",
    importpath = "golang.org/x/sys",
)

go_repository(
    name = "org_golang_x_text",
    importpath = "golang.org/x/text",
    tag = "v0.3.3",
)
