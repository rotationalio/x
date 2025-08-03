#!/usr/bin/env python3

import json

COUNTRIES = "country/testdata/countries.json"


def iso2table():
    with open(COUNTRIES, "r") as f:
        data = json.load(f)

    # Initialize the table as a 26x26 matrix
    table = []
    for i in range(26):
        table.append([""] * 26)

    # Fill the table with ISO 3166-1 alpha-2 codes
    for country in data:
        code = country["iso_3166_1_alpha2"]
        assert len(code) == 2, f"Invalid ISO code: {code}"
        row = ord(code[0]) % 65
        col = ord(code[1]) % 65
        table[row][col] = country["name"]

    print("var iso2table = [][]string{")
    for row in table:
        print('    {"' + '", "'.join(row) + '"},')
    print("}")

    return table


def iso3table():
    with open(COUNTRIES, "r") as f:
        data = json.load(f)

    # Initialize the table as a 26x26 matrix
    table = []
    for i in range(26):
        matrix = []
        for j in range(26):
            matrix.append([""] * 26)
        table.append(matrix)

    # Fill the table with ISO 3166-1 alpha-3 codes
    for country in data:
        code = country["iso_3166_1_alpha3"]
        assert len(code) == 3, f"Invalid ISO code: {code}"
        row = ord(code[0]) % 65
        col = ord(code[1]) % 65
        zen = ord(code[2]) % 65
        table[row][col][zen] = country["name"]

    print("var iso3table = [][][]string{")
    for row in table:
        print('    {')
        for col in row:
            print('        {"' + '", "'.join(col) + '"},')
        print('    },')
    print("}")

    return table


def iso2map():
    with open(COUNTRIES, "r") as f:
        data = json.load(f)

    iso2map = {}
    for country in data:
        code = country["iso_3166_1_alpha2"]
        assert len(code) == 2, f"Invalid ISO code: {code}"
        iso2map[code] = country["name"]

    print("var iso2map = map[string]string{")
    for code, name in iso2map.items():
        print(f'    "{code}": "{name}",')
    print("}")

    return iso2map


def iso3map():
    with open(COUNTRIES, "r") as f:
        data = json.load(f)

    iso3map = {}
    for country in data:
        code = country["iso_3166_1_alpha3"]
        assert len(code) == 3, f"Invalid ISO code: {code}"
        iso3map[code] = country["name"]

    print("var iso3map = map[string]string{")
    for code, name in iso3map.items():
        print(f'    "{code}": "{name}",')
    print("}")

    return iso3map


def char2combos():
    combos = []
    for i in range(26):
        row = []
        for j in range(26):
            row.append(chr(i + 65) + chr(j + 65))
        combos.append(row)

    print("var char2words = []string{")
    for row in combos:
        print('    "' + '", "'.join(row) + '",')
    print("}")


def char3combos():
    combos = []
    for i in range(26):
        for j in range(26):
            row = []
            for k in range(26):
                row.append(chr(i + 65) + chr(j + 65) + chr(k + 65))
            combos.append(row)

    print("var char2words = []string{")
    for row in combos:
        print('    "' + '", "'.join(row) + '",')
    print("}")


if __name__ == "__main__":
    char2combos()
