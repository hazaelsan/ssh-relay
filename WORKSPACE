workspace(name = "ssh_relay")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

# proto_library, cc_proto_library, and java_proto_library rules implicitly
# depend on @com_google_protobuf for protoc and proto runtimes.
# This statement defines the @com_google_protobuf repo.
http_archive(
    name = "com_google_protobuf",
    sha256 = "9510dd2afc29e7245e9e884336f848c8a6600a14ae726adb6befdb4f786f0be2",
    strip_prefix = "protobuf-3.6.1.3",
    urls = ["https://github.com/google/protobuf/archive/v3.6.1.3.zip"],
)

http_archive(
    name = "io_bazel_rules_go",
    urls = ["https://github.com/bazelbuild/rules_go/releases/download/0.16.2/rules_go-0.16.2.tar.gz"],
    sha256 = "f87fa87475ea107b3c69196f39c82b7bbf58fe27c62a338684c20ca17d1d8613",
)

http_archive(
    name = "bazel_gazelle",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.15.0/bazel-gazelle-0.15.0.tar.gz"],
    sha256 = "6e875ab4b6bf64a38c352887760f21203ab054676d9c1b274963907e0768740d",
)

load("@io_bazel_rules_go//go:def.bzl", "go_rules_dependencies", "go_register_toolchains")
go_rules_dependencies()
go_register_toolchains()
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")
gazelle_dependencies()

new_http_archive(
    name = "com_github_google_uuid",
    sha256 = "a02099b9deccb24882b93c817ad8ecd984c7b03e7aacd819c252316c3981b555",
    strip_prefix = "uuid-1.1.0",
    urls = ["https://github.com/google/uuid/archive/v1.1.0.zip"],
    build_file = "bazel/BUILD.uuid",
)

new_http_archive(
    name = "com_github_gorilla_websocket",
    sha256 = "59d1bc909aa6a38de58e5630c48a1fc3089c50f3df5eec73b415c8d51170cd04",
    strip_prefix = "websocket-1.4.0",
    urls = ["https://github.com/gorilla/websocket/archive/v1.4.0.zip"],
    build_file = "bazel/BUILD.websocket",
)

new_http_archive(
    name = "com_github_kylelemons_diff",
    strip_prefix = "godebug-master",
    url = "https://github.com/kylelemons/godebug/archive/master.zip",
    sha256 = "c356a85736d4f0719d8ea59a870827c0aa5acda1ab96fb1ea5562a35e54afb4f",
    build_file = "bazel/BUILD.godebug_diff",
)

new_http_archive(
    name = "com_github_kylelemons_pretty",
    strip_prefix = "godebug-master",
    url = "https://github.com/kylelemons/godebug/archive/master.zip",
    sha256 = "c356a85736d4f0719d8ea59a870827c0aa5acda1ab96fb1ea5562a35e54afb4f",
    build_file = "bazel/BUILD.godebug_pretty",
)
