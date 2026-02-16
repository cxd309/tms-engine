package kinematics

import "math"

// ConstantModelName is the JSON discriminator string for the Constant model.
const ConstantModelName = "constant"

// ConstantAcceleration implements MotionModel using fixed acceleration and deceleration rates.
// This is the default and simplest kinematics model.
//
// JSON discriminator: "model": "constant"
type ConstantAcceleration struct {
	AAcc    float64 `json:"a_acc"` // traction acceleration, m/s²
	ADcc    float64 `json:"a_dcc"` // service braking deceleration, m/s² (positive)
	VMaxVal float64 `json:"v_max"` // maximum speed, m/s
}

func (c ConstantAcceleration) VMax() float64 { return c.VMaxVal }

func (c ConstantAcceleration) BrakingDistance(v float64) float64 {
	if c.ADcc <= 0 {
		return math.Inf(1)
	}
	return (v * v) / (2 * c.ADcc)
}

func (c ConstantAcceleration) BrakingDistanceTo(v, targetV float64) float64 {
	if c.ADcc <= 0 {
		return math.Inf(1)
	}
	if v <= targetV {
		return 0
	}
	return (v*v - targetV*targetV) / (2 * c.ADcc)
}

func (c ConstantAcceleration) VelocityAfterBraking(v0, dist float64) float64 {
	if c.ADcc <= 0 {
		return v0
	}
	return math.Sqrt(math.Max(0, v0*v0-2*c.ADcc*dist))
}

func (c ConstantAcceleration) AccelerateStep(v, targetV, dt float64) (float64, float64) {
	if c.AAcc <= 0 || v >= targetV {
		return targetV * dt, targetV
	}
	tToTarget := (targetV - v) / c.AAcc
	if tToTarget <= dt {
		// Reaches targetV mid-step: accelerate, then cruise for the remainder.
		s1 := v*tToTarget + 0.5*c.AAcc*tToTarget*tToTarget
		s2 := targetV * (dt - tToTarget)
		return s1 + s2, targetV
	}
	newV := v + c.AAcc*dt
	return v*dt + 0.5*c.AAcc*dt*dt, newV
}

func (c ConstantAcceleration) DecelerateStep(v, targetV, dt float64) (float64, float64) {
	if c.ADcc <= 0 || v <= targetV {
		return targetV * dt, targetV
	}
	tToTarget := (v - targetV) / c.ADcc
	if tToTarget <= dt {
		// Reaches targetV mid-step: brake, then cruise for the remainder.
		s1 := v*tToTarget - 0.5*c.ADcc*tToTarget*tToTarget
		s2 := targetV * (dt - tToTarget)
		return math.Max(0, s1) + s2, targetV
	}
	newV := v - c.ADcc*dt
	return math.Max(0, v*dt-0.5*c.ADcc*dt*dt), newV
}
