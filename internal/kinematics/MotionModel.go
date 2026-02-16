// Package kinematics defines the MotionModel interface for vehicle traction and braking
// physics, along with built-in implementations.
//
// Adding a new physics model requires only implementing MotionModel and registering it
// in the JSON discriminator in the service package — the simulation engine itself
// never needs to change.
package kinematics

// MotionModel is the physics contract every kinematics implementation must satisfy.
// All distance values are in metres, velocities in m/s, and time in seconds.
type MotionModel interface {
	// VMax returns the vehicle's maximum permissible speed (m/s).
	VMax() float64

	// BrakingDistance returns the minimum distance needed to stop from velocity v.
	BrakingDistance(v float64) float64

	// BrakingDistanceTo returns the distance needed to decelerate from v to targetV.
	// Returns 0 if v ≤ targetV.
	BrakingDistanceTo(v, targetV float64) float64

	// VelocityAfterBraking returns the velocity reached after braking from v0 over dist metres.
	// Used when a Movement Authority grants less distance than the vehicle proposed.
	VelocityAfterBraking(v0, dist float64) float64

	// AccelerateStep advances the vehicle toward targetV over dt seconds.
	// Handles mid-step transitions: if targetV is reached before dt expires, the
	// vehicle cruises at targetV for the remainder of the timestep.
	// Returns (distance travelled, new velocity).
	AccelerateStep(v, targetV, dt float64) (dist, newV float64)

	// DecelerateStep brakes the vehicle toward targetV (≥ 0) over dt seconds.
	// Handles mid-step transitions: if targetV is reached before dt expires, the
	// vehicle cruises at targetV for the remainder.
	// Returns (distance travelled, new velocity).
	DecelerateStep(v, targetV, dt float64) (dist, newV float64)
}
