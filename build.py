#!/usr/bin/env python3

import argparse
from dataclasses import dataclass
from io import BytesIO
from os.path import join, exists
from platform import machine, system
from threading import Condition, Thread
from subprocess import DEVNULL, call, check_call, check_output
from os import environ, listdir, mkdir, remove
import os
from traceback import print_exc
from typing import Optional
from shutil import which
from zipfile import ZipFile
from requests import get

_version_cache = None


def get_version():
    global _version_cache
    if _version_cache:
        return _version_cache

    ver = None
    try:
        ver = check_output(["git", "describe", "--tags"],
                           stderr=DEVNULL, encoding="utf-8").strip()
    except:
        pass

    if not ver:
        ver = "dev"

    _version_cache = ver
    return ver


_exec_cache: map = {}


def find_executable(name: str, candidates: list) -> Optional[str]:
    if name in _exec_cache:
        return _exec_cache[name]

    found = None
    for candidate in candidates:
        if which(candidate):
            found = candidate
            break
    _exec_cache[name] = found
    return found


def must_find_executable(name: str, candidates: list) -> str:
    res = find_executable(name=name, candidates=candidates)
    if not res:
        raise ValueError(f"Could not find binary for {name}")
    return res

# Based on : https://groups.google.com/d/msg/sage-devel/1lIJ961gV_w/y-2uqPCyzUMJ


def ncpus():
    # for Linux, Unix and MacOS
    if hasattr(os, "sysconf"):
        if "SC_NPROCESSORS_ONLN" in os.sysconf_names:
            #Linux and Unix
            ncpus = os.sysconf("SC_NPROCESSORS_ONLN")
            if isinstance(ncpus, int) and ncpus > 0:
                return ncpus
        else:
            # MacOS X
            return int(os.popen2("sysctl -n hw.ncpu")[1].read())
    # for Windows
    if "NUMBER_OF_PROCESSORS" in os.environ:
        ncpus = int(os.getenv("NUMBER_OF_PROCESSORS", ""))
        if ncpus > 0:
            return ncpus
    # return the default value
    return 1


@dataclass
class Arch():
    name: str
    docker_name: str
    goarch: str
    goenv: map
    upx_supported: bool
    platforms: list


KNOWN_ARCHITECTURES: map = {}
KNOWN_ARCHITECTURE_ALIASES: map = {}


def add_arch(arch: Arch):
    KNOWN_ARCHITECTURES[arch.name] = arch
    if arch.docker_name:
        KNOWN_ARCHITECTURE_ALIASES[arch.docker_name] = arch.name


add_arch(Arch(name="amd64", docker_name="amd64", goarch="amd64",
         upx_supported=True, goenv={}, platforms=["windows", "linux", "darwin"]))
KNOWN_ARCHITECTURE_ALIASES["x86_64"] = "amd64"
add_arch(Arch(name="386", docker_name="i386", goarch="386",
         upx_supported=True, goenv={}, platforms=["windows", "linux"]))

add_arch(Arch(name="arm64", docker_name="arm64", goarch="arm64",
         upx_supported=True, goenv={}, platforms=["windows", "linux", "darwin"]))
KNOWN_ARCHITECTURE_ALIASES["aarch64"] = "arm64"

add_arch(Arch(name="armv5", docker_name="", goarch="arm",
         upx_supported=True, goenv={"GOARM": "5"}, platforms=["linux"]))
add_arch(Arch(name="armv6", docker_name="arm/v6", goarch="arm",
         upx_supported=True, goenv={"GOARM": "6"}, platforms=["linux"]))
add_arch(Arch(name="armv7", docker_name="arm/v7", goarch="arm",
         upx_supported=True, goenv={"GOARM": "7"}, platforms=["linux"]))

add_arch(Arch(name="mips", docker_name="", goarch="mips",
         upx_supported=True, goenv={}, platforms=["linux"]))
add_arch(Arch(name="mips-softfloat", docker_name="", goarch="mips",
         upx_supported=True, goenv={"GOMIPS": "softfloat"}, platforms=["linux"]))
add_arch(Arch(name="mipsle", docker_name="", goarch="mipsle",
         upx_supported=True, goenv={}, platforms=["linux"]))
add_arch(Arch(name="mipsle-softfloat", docker_name="", goarch="mipsle",
         upx_supported=True, goenv={"GOMIPS": "softfloat"}, platforms=["linux"]))
add_arch(Arch(name="mips64", docker_name="", goarch="mips64",
         upx_supported=False, goenv={}, platforms=["linux"]))
add_arch(Arch(name="mips64le", docker_name="", goarch="mips64le",
         upx_supported=False, goenv={}, platforms=["linux"]))


def try_resolve_arch(name: str) -> Optional[str]:
    if name in KNOWN_ARCHITECTURES:
        return name
    if name in KNOWN_ARCHITECTURE_ALIASES:
        return KNOWN_ARCHITECTURE_ALIASES[name]
    return None


def check_call_addenv(args: list, env: map) -> int:
    for k, v in environ.items():
        if k not in env:
            env[k] = v
    return check_call(args, env=env)


def get_local_arch() -> str:
    machine_res = machine().lower()
    if not machine_res:
        raise ValueError("Could not determine local architecture!")
    arch_name = try_resolve_arch(machine_res)
    if not arch_name:
        raise ValueError(
            f"Could not find a supported architecture for: {machine_res}")
    return arch_name


def get_local_platform() -> str:
    system_res = system().lower()
    if not system_res:
        raise ValueError("Could not determine local platform!")
    return system_res


build_task_cond = Condition()


class BuildTask(Thread):
    def __init__(self, dependencies: list, outputs: list, name: str) -> None:
        super().__init__(name=name)
        self.dependencies = dependencies
        self.outputs = outputs
        self.name = name
        self.exc = None

    def can_run(self) -> bool:
        for dep in self.dependencies:
            if not exists(dep):
                return False
        return True

    def _run(self) -> None:
        pass

    def run(self) -> None:
        print(f"Starting: {self.name}", flush=True)
        try:
            self._run()
        except Exception as e:
            self.exc = e
        finally:
            print(f"Done: {self.name}", flush=True)
            build_task_cond.acquire()
            build_task_cond.notify_all()
            build_task_cond.release()

    def join(self, timeout=None):
        super().join(timeout=timeout)
        if self.exc:
            raise self.exc


class GoBuildTask(BuildTask):
    def __init__(self, proj: str, arch: Arch, goos: str, exesuffix: str, cgo: bool, gocmd: str) -> None:
        super().__init__(dependencies=[], outputs=[
            f"dist/{proj}-{goos}-{arch.name}{exesuffix}"], name=f"Go build {proj}-{goos}-{arch.name}{exesuffix}")
        self.arch = arch
        self.goos = goos
        self.proj = proj
        self.cgo = cgo
        self.gocmd = gocmd

    def _run(self) -> None:
        env = {
            "CGO_ENABLED": "1" if self.cgo else "0",
            "GOOS": self.goos,
            "GOARCH": self.arch.goarch,
        }
        for k, v in self.arch.goenv.items():
            env[k] = v

        ldflags = f"-w -s -X 'github.com/Doridian/wsvpn/shared.Version={get_version()}'"
        check_call_addenv([self.gocmd, "build", "-trimpath", "-ldflags",
                          ldflags, "-o", self.outputs[0], f"./{self.proj}"], env=env)


class CompressTask(BuildTask):
    def __init__(self, input: str) -> None:
        super().__init__(dependencies=[input], outputs=[
            f"{input}-compressed"], name=f"UPX {input}")

    def _run(self) -> None:
        check_call(["upx", "-9", f"-o{self.outputs[0]}", self.dependencies[0]])


class DockerBuildTask(BuildTask):
    def __init__(self, gobins: list, tag_latest: bool, push: bool) -> None:
        super().__init__(dependencies=[gobin.outputs[0] for gobin in gobins], outputs=[
        ], name=f"Docker buildx {gobins[0].proj}")

        self.gobins = gobins
        self.push = push
        self.proj = gobins[0].proj

        for gobin in gobins:
            if gobin.goos != "linux":
                raise ValueError("DockerBuildTask is only for Linux targets")
            if gobin.proj != self.proj:
                raise ValueError(
                    "DockerBuildTask can only build one project at a time!")
            if not gobin.arch.docker_name:
                raise ValueError(
                    "Only supply archs to DockerBuildTask that have a valid Docker arch associated!")

        tag_base = f"ghcr.io/doridian/wsvpn/{self.proj}"
        self.tags = [f"{tag_base}:{get_version()}"]
        if tag_latest:
            self.tags.append(f"{tag_base}:latest")

    def _run(self) -> None:
        args = ["docker", "buildx", "build", "--build-arg", f"PROJECT={self.proj}", "--platform", ",".join(
            [f"{gobin.goos}/{gobin.arch.docker_name}" for gobin in self.gobins])]

        for tag in self.tags:
            args.append("-t")
            args.append(tag)

        if self.push:
            args.append("--push")

        args.append(".")

        check_call(args)


class LipoTask(BuildTask):
    def __init__(self, gobins: list) -> None:
        super().__init__(dependencies=[gobin.outputs[0] for gobin in gobins], outputs=[
            f"dist/{gobins[0].proj}-darwin-universal"], name=f"Lipo {gobins[0].proj}")

        self.gobins = gobins
        self.proj = gobins[0].proj

        for gobin in gobins:
            if gobin.goos != "darwin":
                raise ValueError("LipoTask is only for Darwin targets")
            if gobin.proj != self.proj:
                raise ValueError(
                    "LipoTask can only build one project at a time!")

    def _run(self) -> None:
        from slimfat import make_fat
        make_fat(self.outputs[0], [gobin.outputs[0] for gobin in self.gobins])


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--platforms", "--platform", "-p", default="*", required=False, type=str,
                        help="Which platforms to build for (* for all, local for host machine, comma separated). Accepted: linux, darwin, windows")
    parser.add_argument("--architectures", "--architecture", "-a", default="*", required=False, type=str,
                        help="Which architectures to build for (* for all, local for host machine, comma separated). Use \"list\" to get a list")
    parser.add_argument("--projects", "--project", "-i", default="*", required=False, type=str,
                        help="Which projects to build (* for all, comma separated). Accepted: client, server, wsvpn")
    parser.add_argument("--compress", "-c", default=False, action="store_true",
                        help="Output UPX compressed binaries for Linux")
    parser.add_argument("--lipo", default=False, action="store_true",
                        help="Produce universal binaries using lipo or llvm-lipo")
    parser.add_argument("--docker", default=False, action="store_true",
                        help="Whether to build Docker images for Linux")
    parser.add_argument("--docker-tag-latest", default=False, action="store_true",
                        help="Whether to tag latest on built Docker images")
    parser.add_argument("--docker-push", default=False, action="store_true",
                        help="Whether to push Docker images to the registry")
    parser.add_argument("--jobs", "-j", default=ncpus(),
                        type=int, help="How many jobs to run in parallel")
    parser.add_argument("--cgo", default=False, action="store_true",
                        help="Will enable CGO in all builds")
    parser.add_argument("--gocmd", default="go", type=str,
                        help="Use this command instead of go to build")
    parser.add_argument("--download-wintun", default=False, action="store_true", help="Download wintun DLLs")
    parser.add_argument("--no-download-wintun", default=False, action="store_true", help="Never download wintun DLLs")
    flags = parser.parse_args()

    platforms = None
    if flags.platforms == "*":
        platforms = ["linux", "darwin", "windows"]
    elif flags.platforms == "local":
        platforms = [get_local_platform()]
    else:
        platforms = flags.platforms.split(",")

    projects = None
    if flags.projects == "*":
        projects = ["client", "server", "wsvpn"]
    else:
        projects = flags.projects.split(",")

    architectures = None
    if flags.architectures == "*":
        architectures = [arch for arch in KNOWN_ARCHITECTURES]
    elif flags.architectures == "local":
        architectures = [get_local_arch()]
    elif flags.architectures == "list":
        print("Supported architectures:", flush=True)
        for _, arch in KNOWN_ARCHITECTURES.items():
            print(f"\t- {arch.name} (on {', '.join(arch.platforms)})", flush=True)
        return
    else:
        architectures = [try_resolve_arch(arch)
                         for arch in flags.architectures.split(",")]

    print(f"Building version: {get_version()}", flush=True)

    print("Cleaning dist...", flush=True)
    try:
        mkdir("dist")
    except FileExistsError:
        pass
    for distfile in listdir("dist"):
        remove(join("dist", distfile))

    print("Generating all build tasks...", flush=True)
    tasks: list = []
    task_types: set = set()
    task_platforms: set = set()
    def addTask(task: BuildTask) -> None:
        tasks.append(task)
        task_types.add(type(task))

    for proj in projects:
        if not proj:
            continue
        for platform in platforms:
            exesuffix = ""
            if platform == "windows":
                exesuffix = ".exe"

            platform_tasks: list = []
            for arch_name in architectures:
                arch = KNOWN_ARCHITECTURES[arch_name]
                if platform not in arch.platforms:
                    continue

                task = GoBuildTask(proj=proj, arch=arch, goos=platform,
                                   exesuffix=exesuffix, cgo=flags.cgo, gocmd=flags.gocmd)
                platform_tasks.append(task)

                addTask(task)
                if flags.compress and platform == "linux" and task.arch.upx_supported:
                    addTask(CompressTask(input=task.outputs[0]))

            if not platform_tasks:
                continue

            task_platforms.add(platform)

            if platform == "linux" and flags.docker:
                addTask(DockerBuildTask([task for task in platform_tasks if task.arch.docker_name],
                            tag_latest=flags.docker_tag_latest, push=flags.docker_push))

            if platform == "darwin" and flags.lipo:
                addTask(
                    LipoTask([task for task in platform_tasks if task.goos == "darwin"]))

    if GoBuildTask in task_types:
        print("Downloading Go modules...", flush=True)
        check_call([flags.gocmd, "mod", "download"])

    if DockerBuildTask in task_types:
        print("Preparing Docker buildx...", flush=True)
        call(["docker", "buildx", "create", "--name", "multiarch"],
             stdout=DEVNULL, stderr=DEVNULL)
        check_call(["docker", "buildx", "use", "multiarch"])

    if ("windows" in task_platforms and not flags.no_download_wintun) or flags.download_wintun:
        url = "https://www.wintun.net/builds/wintun-0.14.1.zip"
        print(f"Downloading WinTun from \"{url}\"...", flush=True)

        with BytesIO() as stream:
            res = get(url)
            res.raise_for_status()
            stream.write(res.content)
            stream.flush()

            outdir = "shared/iface/wintun"
            try:
                mkdir(outdir)
            except FileExistsError:
                pass
            zip = ZipFile(stream)
            zip.extractall(outdir)

    print("Executing build tasks...", flush=True)

    def pick_task() -> Optional[BuildTask]:
        for i, task in enumerate(tasks):
            if task.can_run():
                return tasks.pop(i)

        return None

    all_tasks: list = tasks.copy()

    parallelism_allowed = False
    num_jobs = 1
    running_tasks: list = []
    while len(tasks) > 0:
        while len(running_tasks) < num_jobs:
            task = pick_task()
            if not task:
                break
            running_tasks.append(task)
            task.start()
            if not parallelism_allowed and isinstance(task, GoBuildTask):
                task.join()
                parallelism_allowed = True
                num_jobs = flags.jobs
                running_tasks = []

        if len(running_tasks) > 0:
            build_task_cond.acquire()
            build_task_cond.wait()
            build_task_cond.release()
        else:
            break

        running_tasks = [task for task in running_tasks if task.is_alive()]

    had_errors = False

    for task in all_tasks:
        try:
            task.join()
        except Exception:
            print(f"Error: {task.name}", flush=True)
            print_exc()
            had_errors = True

    if had_errors:
        raise Exception("One or more tasks had errors!")

    if len(tasks) > 0:
        raise Exception("Could not start all tasks...")

    print("Build OK!", flush=True)


if __name__ == "__main__":
    main()
