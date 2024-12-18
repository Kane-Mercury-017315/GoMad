package main

import (
	"math"
)

// a1: the atom 1
// a2: the atom 2
// epsilon: the vacuum dielectric permittivity
// r: distance between q1 and q2
func CalculateElectricPotentialEnergy(a1, a2 *Atom, r float64) float64 {
	chargeMagnitude := a1.charge * a2.charge

	if chargeMagnitude < 0.0 {
		return -chargeMagnitude / (4 * math.Pi * epsilon * r)
	}

	return chargeMagnitude / (4 * math.Pi * epsilon * r)
}

// A: coefficient 1
// B: coefficient 2
// r: distance between atom 1 and atom 2
func CalculateLJPotentialEnergy(B, A, r float64) float64 {
	r_6 := math.Pow(r, 6)
	r_12 := r_6 * r_6
	LJ := (A / r_12) - (B / r_6)
	if LJ < 0 {
		return -LJ
	}
	return LJ
}

func NewVerletList() *VerletList {
	return &VerletList{
		Neighbors: make(map[*Atom][]*Atom),
		Cutoff:    verletCutOff,
		Buffer:    verletBuffer,
	}
}

func (v *VerletList) BuildVerlet(protein *Protein) {
	cutoffPlusBuffer := v.Cutoff + v.Buffer
	v.Neighbors = make(map[*Atom][]*Atom)

	for _, residue := range protein.Residue {
		for _, atom := range residue.Atoms {
			v.Neighbors[atom] = []*Atom{}
			for _, otherResidue := range protein.Residue {
				for _, otherAtom := range otherResidue.Atoms {
					if atom == otherAtom {
						continue
					}
					// Exclude atoms within 3 bonds
					if otherAtom.index >= atom.index-3 && otherAtom.index <= atom.index+3 {
						continue
					}
					distance := Distance(atom.position, otherAtom.position)
					if distance <= cutoffPlusBuffer {
						v.Neighbors[atom] = append(v.Neighbors[atom], otherAtom)
					}
				}
			}
		}
	}
}

func (protein *Protein) AssignChargesToProtein(chargeData map[string]map[string]float64) {
	for _, residue := range protein.Residue {
		residueName := residue.Name

		// Get the charge data for this residue, if it exists
		residueChargeData, residueExists := chargeData[residueName]

		for _, atom := range residue.Atoms {
			atomName := atom.element

			if residueExists {
				// Try to get the charge data for this atom
				atomCharge, atomExists := residueChargeData[atomName]
				if atomExists {
					// Assign the charge from the charge data
					atom.charge = atomCharge
					continue
				}
			}

			// If there's no data for this atom or residue, assign a charge of 0
			atom.charge = 0.0
		}
	}
}

func CalculateTotalUnbondedEnergyForce(p *Protein, nonbondedParameter parameterDatabase) (float64, map[int]*TriTuple) {
	forceMap := make(map[int]*TriTuple)
	totalEnergy := 0.0
	verletList := NewVerletList()
	verletList.BuildVerlet(p)

	for _, residue := range p.Residue {
		for _, atom1 := range residue.Atoms {
			// Initialize force for atom1
			forceMap[atom1.index] = &TriTuple{0.0, 0.0, 0.0}

			// Access the Neighbors map using the dereferenced verletList
			neighbors, exists := verletList.Neighbors[atom1]
			if !exists {
				continue
			}

			for _, atom2 := range neighbors {
				// Compute the distance between atom1 and atom2
				r := Distance(atom1.position, atom2.position)

				// Calculate the Lennard-Jones potential energy between atom1 and atom2
				parameterList := SearchParameter(2, nonbondedParameter, atom1, atom2)
				if len(parameterList) == 2 {
					LJPotentialEnergy := CalculateLJPotentialEnergy(parameterList[0], parameterList[1], r)
					totalEnergy += LJPotentialEnergy
					// Calculate the Lennard-Jones force between atom1 and atom2
					LJForce := CalculateLJForce(atom1, atom2, parameterList[0], parameterList[1], r)
					// Update the force map for atom1
					forceMap[atom1.index].x += LJForce.x
					forceMap[atom1.index].y += LJForce.y
					forceMap[atom1.index].z += LJForce.z
				}

				if atom1.charge == 0.0 || atom2.charge == 0.0 {
					continue
				}

				// Calculate the electric potential energy between atom1 and atom2
				electricPotentialEnergy := CalculateElectricPotentialEnergy(atom1, atom2, r)
				totalEnergy += electricPotentialEnergy
				// Calculate the electric force between atom1 and atom2
				electricForce := CalculateElectricForce(atom1, atom2, r)

				// Update the force map for atom1
				forceMap[atom1.index].x += electricForce.x
				forceMap[atom1.index].y += electricForce.y
				forceMap[atom1.index].z += electricForce.z
			}
		}
	}

	return totalEnergy, forceMap
}

func CalculateElectricForce(a1, a2 *Atom, r float64) TriTuple {
	chargeMagnitude := a1.charge * a2.charge

	forceMagnitude := 0.0
	if chargeMagnitude > 0.0 {
		forceMagnitude = chargeMagnitude / (4 * math.Pi * epsilon * r * r)
	} else {
		forceMagnitude = -chargeMagnitude / (4 * math.Pi * epsilon * r * r)
	}

	unitVector := TriTuple{
		x: (a2.position.x - a1.position.x) / r,
		y: (a2.position.y - a1.position.y) / r,
		z: (a2.position.z - a1.position.z) / r,
	}
	return TriTuple{
		x: forceMagnitude * unitVector.x,
		y: forceMagnitude * unitVector.y,
		z: forceMagnitude * unitVector.z,
	}
}

func CalculateLJForce(a1, a2 *Atom, B, A, r float64) TriTuple {
	r_6 := math.Pow(r, 6)
	r_12 := r_6 * r_6

	forceMagnitude := (-12*A/r_12 + 6*B/r_6) / r
	unitVector := TriTuple{
		x: (a2.position.x - a1.position.x) / r,
		y: (a2.position.y - a1.position.y) / r,
		z: (a2.position.z - a1.position.z) / r,
	}
	return TriTuple{
		x: forceMagnitude * unitVector.x,
		y: forceMagnitude * unitVector.y,
		z: forceMagnitude * unitVector.z,
	}
}
