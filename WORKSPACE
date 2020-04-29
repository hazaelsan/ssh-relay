workspace(name = "ssh_relay")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

# Bazel core.
http_archive(
    name = "rules_cc",
    sha256 = "35f2fb4ea0b3e61ad64a369de284e4fbbdcdba71836a5555abb5e194cf119509",
    strip_prefix = "rules_cc-624b5d59dfb45672d4239422fa1e3de1822ee110",
    urls = [
        "https://github.com/bazelbuild/rules_cc/archive/624b5d59dfb45672d4239422fa1e3de1822ee110.tar.gz",
    ],
)

# rules_java defines rules for generating Java code from Protocol Buffers.
http_archive(
    name = "rules_java",
    sha256 = "ccf00372878d141f7d5568cedc4c42ad4811ba367ea3e26bc7c43445bbc52895",
    strip_prefix = "rules_java-d7bf804c8731edd232cb061cb2a9fe003a85d8ee",
    urls = [
        "https://github.com/bazelbuild/rules_java/archive/d7bf804c8731edd232cb061cb2a9fe003a85d8ee.tar.gz",
    ],
)

# rules_proto defines abstract rules for building Protocol Buffers.
http_archive(
    name = "rules_proto",
    sha256 = "2490dca4f249b8a9a3ab07bd1ba6eca085aaf8e45a734af92aad0c42d9dc7aaf",
    strip_prefix = "rules_proto-218ffa7dfa5408492dc86c01ee637614f8695c45",
    urls = [
        "https://github.com/bazelbuild/rules_proto/archive/218ffa7dfa5408492dc86c01ee637614f8695c45.tar.gz",
    ],
)

load("@rules_cc//cc:repositories.bzl", "rules_cc_dependencies")

rules_cc_dependencies()

load("@rules_java//java:repositories.bzl", "rules_java_dependencies", "rules_java_toolchains")

rules_java_dependencies()

rules_java_toolchains()

load("@rules_proto//proto:repositories.bzl", "rules_proto_dependencies", "rules_proto_toolchains")

rules_proto_dependencies()

rules_proto_toolchains()

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
    name = "bazel_federation",
    sha256 = "9d4fdf7cc533af0b50f7dd8e58bea85df3b4454b7ae00056d7090eb98e3515cc",
    strip_prefix = "bazel-federation-130c84ec6d60f31b711400e8445a8d0d4a2b5de8",
    type = "zip",
    url = "https://github.com/bazelbuild/bazel-federation/archive/130c84ec6d60f31b711400e8445a8d0d4a2b5de8.zip",
)

load("@bazel_federation//:repositories.bzl", "rules_python")

rules_python()

load("@bazel_federation//setup:rules_python.bzl", "rules_python_setup")

rules_python_setup()

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "7b9bbe3ea1fccb46dcfa6c3f3e29ba7ec740d8733370e21cdc8937467b4a4349",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.22.4/rules_go-v0.22.4.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.22.4/rules_go-v0.22.4.tar.gz",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "d8c45ee70ec39a57e7a05e5027c32b1576cc7f16d9dd37135b0eddde45cf1b10",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/bazel-gazelle/releases/download/v0.20.0/bazel-gazelle-v0.20.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.20.0/bazel-gazelle-v0.20.0.tar.gz",
    ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies()

# Deps.
http_archive(
    name = "com_github_golang_glog",
    build_file = "bazel/BUILD.golang_glog",
    sha256 = "433e5b9696e71828a109cef978312d650dd78f51eb3fd4dc59656013932dd0d1",
    strip_prefix = "glog-23def4e6c14b4da8ac2ed8007337bc5eb5007998",
    url = "https://github.com/golang/glog/archive/23def4e6c14b4da8ac2ed8007337bc5eb5007998.zip",
)

http_archive(
    name = "com_github_google_uuid",
    build_file = "bazel/BUILD.uuid",
    sha256 = "b2041c5847c227bb5feee5fa5aaa32736e094a2d26357e66fcc5cc405c36386a",
    strip_prefix = "uuid-1.1.1",
    urls = ["https://github.com/google/uuid/archive/v1.1.1.zip"],
)

http_archive(
    name = "com_github_gorilla_websocket",
    build_file = "bazel/BUILD.websocket",
    sha256 = "335a84a456112cce890d00ec82e59cfe5a07581f53bd0b9284d21714d0527bc1",
    strip_prefix = "websocket-1.4.2",
    urls = ["https://github.com/gorilla/websocket/archive/v1.4.2.zip"],
)

http_archive(
    name = "com_github_kylelemons_diff",
    build_file = "bazel/BUILD.godebug_diff",
    sha256 = "a07edfa7b01c277196479e1ec51b92b416f2935c049f96917632e9c000e146f8",
    strip_prefix = "godebug-1.1.0",
    url = "https://github.com/kylelemons/godebug/archive/v1.1.0.zip",
)

http_archive(
    name = "com_github_kylelemons_pretty",
    build_file = "bazel/BUILD.godebug_pretty",
    sha256 = "a07edfa7b01c277196479e1ec51b92b416f2935c049f96917632e9c000e146f8",
    strip_prefix = "godebug-1.1.0",
    url = "https://github.com/kylelemons/godebug/archive/v1.1.0.zip",
)
