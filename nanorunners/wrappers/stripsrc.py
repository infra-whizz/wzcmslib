#!/usr/bin/python3

import sys
import token
import tokenize

def main(fname):
    """
    Strip python source (empty lines, comments, docstrings)
    """
    source = open(fname)
    dst = []

    pt = token.INDENT
    ln = -1
    lcol = 0

    tokgen = tokenize.generate_tokens(source.readline)
    for t, data, (sn, scol), (en, ecol), _ in tokgen:
        if sn > ln:
            lcol = 0
        if scol > lcol:
            dst.append(" " * (scol - lcol))
        if t == token.STRING and (pt == token.INDENT or pt == token.NEWLINE) or t == tokenize.COMMENT:
            continue
        else:
            dst.append(data)
        pt = t
        lcol = ecol
        ln = en

    for l in filter(None, "".join(dst).split("\n")):
        if l.strip():
            print(l)

if __name__ == '__main__':
    main(sys.argv[1])
