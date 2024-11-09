package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// /////////////////
// ////////////////
// ///////////////
// ////These function are used for read protein from PDB

// Function to parse a PDB line based on spaces
func parsePDBLine(line string) (Atom, string, error) {
	fields := strings.Fields(line)
	var atom Atom

	// Parse coordinates
	x, err := strconv.ParseFloat(fields[6], 64)
	if err != nil {
		return Atom{}, "", fmt.Errorf("error parsing x position: %v", err)
	}
	y, err := strconv.ParseFloat(fields[7], 64)
	if err != nil {
		return Atom{}, "", fmt.Errorf("error parsing y position: %v", err)
	}
	z, err := strconv.ParseFloat(fields[8], 64)
	if err != nil {
		return Atom{}, "", fmt.Errorf("error parsing z position: %v", err)
	}

	// Parse element symbol
	element := fields[2]
	index, _ := strconv.Atoi(fields[1])
	// pass value to atom object
	atom.position.x = x
	atom.position.y = y
	atom.position.z = z
	atom.element = element
	atom.index = index
	// Extract residue name
	residueName := fields[3]

	return atom, residueName, nil
}

func readProteinFromFile(fileName string) (Protein, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return Protein{}, err
	}
	defer file.Close()

	var residues []*Residue
	var currentResidue *Residue
	var protein Protein

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Process lines that start with "ATOM"
		if strings.HasPrefix(line, "ATOM") {
			atom, residueName, err := parsePDBLine(line)
			if err != nil {
				return Protein{}, err
			}

			// If it's a new residue or first row, start a new Residue object
			if currentResidue == nil || currentResidue.Name != residueName {
				// Add the previous residue to the list if it exists
				if currentResidue != nil {
					residues = append(residues, currentResidue)
				}

				// otherwise,Create a new Residue object
				currentResidue = &Residue{
					Name:  residueName,
					Atoms: []*Atom{&atom},
				}
			} else {
				// If it's the same residue, add the atom to the current residue
				currentResidue.Atoms = append(currentResidue.Atoms, &atom)
			}
		}
	}

	// Add the last residue to the list
	if currentResidue != nil {
		residues = append(residues, currentResidue)
	}

	if err := scanner.Err(); err != nil {
		return Protein{}, err
	}

	// Set the residues in the protein
	protein.Residue = residues
	protein.UpdateMasses(massTable)

	return protein, nil
}

func (p *Protein) UpdateMasses(massTable map[string]float64) {
	for _, residue := range p.Residue {
		for _, atom := range residue.Atoms {
			// Extract the first character of the element to match in the mass table
			baseElement := string(atom.element[0])

			if mass, found := massTable[baseElement]; found {
				atom.mass = mass
			} else {
				fmt.Printf("Warning: Mass not found for element %s (using base element %s)\n", atom.element, baseElement)
				atom.mass = 0.0 //
			}
		}
	}
}

var massTable = map[string]float64{
	"H": 1.0079,
	"C": 12.0107,
	"N": 14.0067,
	"O": 15.9994,
	"S": 32.065,
	// Add more elements as needed
}

// /////////////////
// ////////////////
// ///////////////
// ////These function are used for read parameter for MDsimulation

// Function to parse a single line and return a parameterPair struct
func ParseParameterPairLine(line string, funcPosition, length int) (parameterPair, error) {
	// Remove any leading/trailing whitespace
	line = strings.TrimSpace(line)

	// Skip empty lines or lines that start with a comment
	if line == "" || strings.HasPrefix(line, ";") {
		return parameterPair{}, fmt.Errorf("empty or comment line")
	}

	// check the number of line
	fields := strings.Fields(line)
	// Initialize parameterPair struct and parse atom names
	atomName := fields[:funcPosition-1]
	parameter := make([]float64, length-funcPosition-1)

	var pair parameterPair
	pair.atomName = atomName
	pair.parameter = parameter

	// Parse function
	function, err := strconv.Atoi(fields[funcPosition-1])
	if err != nil {
		return parameterPair{}, err
	}
	pair.Function = function

	// Parse parameters
	for i := 0; i < len(pair.parameter); i++ {
		param, err := strconv.ParseFloat(fields[funcPosition+i], 64)
		if err != nil {
			return parameterPair{}, err
		}
		pair.parameter[i] = param
	}

	return pair, nil
}

// Function to read the entire file and parse each line into a slice of parameterPair structs
func ReadParameterFile(filePath string) (parameterDatabase, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return parameterDatabase{}, err
	}
	defer file.Close()

	var pairs parameterDatabase
	Firstline, _ := GetFirstLine(filePath)
	funcPosition, len, _ := FindPosition(Firstline)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		pair, err := ParseParameterPairLine(line, funcPosition, len)
		if err != nil {
			continue
		}
		pairPointer := &pair
		pairs.atomPair = append(pairs.atomPair, pairPointer)
	}

	if err := scanner.Err(); err != nil {
		return parameterDatabase{}, err
	}

	return pairs, nil
}

// Function to read the entire file and parse each line into a slice of parameterPair structs
func FindPosition(line string) (int, int, error) {
	// Check if the line starts with ";"
	if !strings.HasPrefix(line, ";") {
		return -1, 0, fmt.Errorf("line does not start with a comment: %s", line)
	}

	// Split the line into fields
	fields := strings.Fields(line)
	len := len(fields)

	// Look for "func" in the fields
	for i, field := range fields {
		if field == "func" {
			// Return the index of "func"
			return i, len, nil
		}
	}

	return -1, 0, fmt.Errorf("'func' not found in the line")
}

func GetFirstLine(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Read and return the first line
	if scanner.Scan() {
		return scanner.Text(), nil
	}

	return "", fmt.Errorf("file does not have any lines")
}

// // This part read the aminoacids.rtp
func ReadAminoAcidsPara(fileName string) (map[string]residueParameter, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	residues := make(map[string]residueParameter)
	var currentResidue *residueParameter
	section := ""

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.Trim(line, "[] ")
			if section != "atoms" && section != "bonds" && section != "angles" && section != "dihedrals" && section != "impropers" &&
				section != "all_dihedrals" && section != "HH14" && section != "RemoveDih" && section != "bondedtypes" {
				currentResidue = &residueParameter{name: section}
				residues[section] = *currentResidue
			}
			continue
		}

		if currentResidue == nil {
			continue
		}

		switch section {
		case "atoms":
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				x, _ := strconv.ParseFloat(parts[2], 64)
				y, _ := strconv.ParseFloat(parts[3], 64)
				currentResidue.atoms = append(currentResidue.atoms, &atoms{
					atoms: []string{parts[0], parts[1]},
					x:     x,
					y:     y,
				})
			}
		case "bonds":
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				currentResidue.bonds = append(currentResidue.bonds, &bonds{
					atoms: []string{parts[0], parts[1]},
					para:  parts[2],
				})
			}
		case "angles":
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				currentResidue.angles = append(currentResidue.angles, &angles{
					atoms:       []string{parts[0], parts[1], parts[2]},
					gromos_type: parts[3],
				})
			}
		case "dihedrals":
			parts := strings.Fields(line)
			if len(parts) >= 5 {
				currentResidue.dihedrals = append(currentResidue.dihedrals, &dihedrals{
					atoms:       []string{parts[0], parts[1], parts[2], parts[3]},
					gromos_type: parts[4],
				})
			}
		}
		residues[currentResidue.name] = *currentResidue
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return residues, nil
}

func parseChargeFile(filename string) (map[string]map[string]AtomChargeData, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	chargeData := make(map[string]map[string]AtomChargeData)
	scanner := bufio.NewScanner(file)
	var currentResidue string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		// Check for residue header lines like "[ ALA ]"
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentResidue = strings.TrimSpace(line[1 : len(line)-1])
			chargeData[currentResidue] = make(map[string]AtomChargeData)
			continue
		}

		// Parse atom data lines
		fields := strings.Fields(line)

		// Ensure that we have exactly four columns
		if len(fields) != 4 {
			return nil, fmt.Errorf("invalid line format: %s", line)
		}

		atomName := fields[0]
		atomType := fields[1]
		atomChargeStr := fields[2]
		chargeGroupStr := fields[3]

		// Parse atom charge
		atomCharge, err := strconv.ParseFloat(atomChargeStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid atom charge '%s' in line: %s", atomChargeStr, line)
		}

		// Parse charge group (can be integer)
		chargeGroup, err := strconv.Atoi(chargeGroupStr)
		if err != nil {
			return nil, fmt.Errorf("invalid charge group '%s' in line: %s", chargeGroupStr, line)
		}

		// Store the charge data
		if currentResidue == "" {
			return nil, fmt.Errorf("atom data without residue header: %s", line)
		}
		chargeData[currentResidue][atomName] = AtomChargeData{
			AtomType:    atomType,
			AtomCharge:  atomCharge,
			ChargeGroup: chargeGroup,
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return chargeData, nil
}
