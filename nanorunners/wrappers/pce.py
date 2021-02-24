#!/usr/bin/python3
#
# Ansible module runner behind chroot without Python installed.
# Author: Bo Maryniuk <bogdan.maryniuk@elektrobit.com>

import os
import sys
import argparse
import importlib
import shutil
import re


class ChrootCaller:
    """
    Ansible modules adaptor that calls modules inside the chroot without Python installed.
    """

    # List of basic pre-loaded modules, required by most Ansible plugins
    DEFAULT_MODULES:set = {
        "math",
        "fcntl",
        "_posixsubprocess",
        "select",
        "_random",
        "_sha1",
        "_sha256",
        "_sha3",
        "_sha512",
        "_blake2",
        "_md5",
    }

    def __init__(self, args):
        self.args = args
        self.mod:str = ""
        self.syspath = []
        self.venv = "/tmp/.waka/.venv"
        self.dynlibs:list = []

    def dynmod_update_loadlist(self) -> None:
        """
        Merge modules from the external list
        """
        try:
            for mod in open(self.args.modules).readlines():
                mod = mod.strip()
                if not mod: continue
                self.DEFAULT_MODULES.add(mod)
        except Exception as err:
            sys.stderr.write("ERROR: Unable to access list of extra modules to load: {}\n".format(err))
            sys.exit(1)

    def dynmod_to_name(self, root:str, dynmod:str) -> str:
        """
        Convert module path to an importable spacename
        """
        if dynmod.startswith(root):
            dynmod = dynmod[len(root):]
        if "site-packages" in dynmod:
            dynmod = dynmod.split("site-packages")[-1]
        else:
            dynmod = dynmod.split("lib-dynload")[-1]
        dp = list(filter(None, dynmod.split("/")))
        dp[-1] = dp[-1].split(".")[0]
        dynmod = ".".join(dp)

        return dynmod

    def dynmod_find(self) -> None:
        """
        Load dynamic modules from the base Python distribution.
        """
        for p in self.syspath:
            for r, _, f in os.walk(p, topdown=False):
                for fn in f:
                    if fn.endswith(".so"):
                        self.dynlibs.append(self.dynmod_to_name(p, "/".join([r, fn])))
        self.dynlibs = sorted(self.dynlibs)

    def dynmod_preload(self) -> None:
        """
        Preload dynamically linked modules.
        They will stay opened across the chroot.
        """
        for mod in self.DEFAULT_MODULES:
            try:
                importlib.import_module(mod)
            except ImportError as exc:
                sys.stderr.write("Error: {}\n".format(exc))
                sys.exit(1)

    def clone_pylib(self) -> None:
        """
        Copy all python libraries for temporary use
        """
        o_venv = re.sub("/+", "/", self.args.root + self.venv)

        # Get distinct search paths
        # Should skip if done already. file-based marker?
        here = os.path.abspath(".")
        sys.path.sort(key=len)
        for p in sys.path:
            ex = False
            if not p: continue
            for pr in self.syspath:
                if p.startswith(pr) or p.startswith(here):
                    ex = True
            if not ex and p not in self.syspath and os.path.isdir(p):
                self.syspath.append(p)

        # copy tree
        if not os.path.isdir(o_venv):
            for p in self.syspath:
                #print("copying everything from {} to {}".format(p, o_venv + p))
                # TODO: filter-out rubbish (cache, pyc, pyo, txt)
                shutil.copytree(p, o_venv + p)

    def update_syspath(self):
        """
        Drop bogus search places after chroot.
        """
        # Add main entry points paths, those are /usr/lib64/python... and /usr/lib/python.../site-packages
        # and remap them on the remote chroot inside the temporary venv root
        for sp in self.syspath:
            sys.path.append(self.venv + sp)

        fsp = []
        for p in sys.path:
            if os.path.isdir(p):
                fsp.append(p)
        sys.path = fsp[:]

        # Reimport all the current local modules from chrooted venv so they
        # won't conflict with anything external
        from importlib import reload
        for m in ["os", "sys", "argparse", "shutil", "re", "importlib"]:
            reload(sys.modules[m])

    def setup(self) -> "ChrootCaller":
        """
        Setup the environment
        """
        if self.args.modules:
            self.dynmod_update_loadlist()

        self.dynmod_preload()

        sloc = os.path.dirname(self.args.cmd)
        if sloc:
            sys.path.append(sloc)
        self.mod = os.path.basename(self.args.cmd).split(".")[0]
        if self.mod in sys.modules:
            sys.stderr.write("Error: module {} is clashing with already imported as {}\n"
                             .format(self.mod, sys.modules[self.mod].__file__))
            sys.exit(1)

        self.dynmod_find()

        # Only display avilable module
        if self.args.list_modules:
            for mod in self.dynlibs:
                print(mod)
            sys.exit(1)

        self.clone_pylib()

        return self

    def invoke(self) -> None:
        """
        Import specified module
        """
        self.update_syspath()
        try:
            importlib.import_module(self.mod)
        except ImportError as err:
            sys.stderr.write("Error running module {}: {}\n".format(self.mod, err))
            sys.exit(1)

        getattr(sys.modules[self.mod], self.args.func)()

    def run(self) -> None:
        """
        Run an Ansible module
        """
        pid = os.fork()

        uid, gid = os.getuid(), os.getgid()

        if not pid:
            os.setuid(uid)
            os.chroot(self.args.root)

            self.invoke()
            sys.exit(0)


def main() -> None:
    """
    """
    ap = argparse.ArgumentParser("pce", description="Python Chroot Executor - Python inside chroot without being preinstalled")
    ap.add_argument("-r", "--root", type=str, help="Root of the directory where to chroot")
    ap.add_argument("-c", "--cmd", type=str, help="Command line after the chroot")
    ap.add_argument("-f", "--func", type=str, help="Function name after import", default="main")
    ap.add_argument("-m", "--modules", type=str, help="Path to a text file for dynamically linked .so modules (one line per a name)")
    ap.add_argument("-l", "--list-modules", action="store_true", help="List all available dynamically linked modules")
    args = ap.parse_args()

    if len(sys.argv) < 2 or not args.cmd:
        ap.print_usage()
        sys.exit(1)
    else:
        ChrootCaller(args).setup().run()

if __name__ == "__main__":
    main()
