/*
   Copyright 2023 The Kubernetes Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

// Package prescore exports an api.PreScorePlugin to the host. Only import this
// package when setting Plugin, as doing otherwise will cause overhead.
package prescore

import (
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/api"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/cyclestate"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/imports"
	"sigs.k8s.io/kube-scheduler-wasm-extension/guest/internal/plugin"
	protoapi "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/api"
	meta "sigs.k8s.io/kube-scheduler-wasm-extension/kubernetes/proto/meta"
)

// prescore is the current plugin assigned with SetPlugin.
var prescore api.PreScorePlugin

// SetPlugin should be called in `main` to assign an api.PreScorePlugin
// instance.
//
// For example:
//
//	func main() {
//		plugin := scorePlugin{}
//		prescore.SetPlugin(plugin)
//		score.SetPlugin(plugin)
//	}
//
//	type scorePlugin struct{}
//
//	func (scorePlugin) PreScore(state api.CycleState, pod proto.Pod, nodeList proto.NodeList) *api.Status {
//		// Write state you need on Score
//		return nil
//	}
//
//	func (scorePlugin) Score(state api.CycleState, pod proto.Pod, nodeName string) (int32, *api.Status) {
//		var score int32
//		// Derive score for the node name using state set on PreScore!
//		return score, nil
//	}
//
// Note: This should only be set when score.SetPlugin also is.
func SetPlugin(preScorePlugin api.PreScorePlugin) {
	if preScorePlugin == nil {
		panic("nil PreScorePlugin")
	}
	prescore = preScorePlugin
	plugin.MustSet(prescore)
}

// prevent unused lint errors (lint is run with normal go).
var _ func() uint32 = _prescore

// prescore is only exported to the host.
//
//export prescore
func _prescore() uint32 {
	if prescore == nil {
		// If we got here, someone imported the package, but forgot to set the
		// filter. Panic with what's wrong.
		panic("prescore imported, but prescore.SetPlugin not called")
	}

	// Pod is lazy and the same value for all plugins in a scheduling cycle.
	pod := cyclestate.Pod

	s := prescore.PreScore(cyclestate.Values, pod, &nodeList{})

	return imports.StatusToCode(s)
}

// nodeList is lazy so that a plugin which doesn't read fields avoids a
// relatively expensive unmarshal penalty.
//
// Note: Unlike proto.Pod, this is not special cased for the scheduling cycle.
type nodeList struct {
	nl *protoapi.NodeList
}

func (n *nodeList) Metadata() *meta.ListMeta {
	return n.lazyNodeList().Metadata
}

func (n *nodeList) Items() []*protoapi.Node {
	return n.lazyNodeList().Items
}

// lazyNodeList lazy initializes node from imports.NodeList.
func (n *nodeList) lazyNodeList() *protoapi.NodeList {
	if nl := n.nl; nl != nil {
		return nl
	}

	var msg protoapi.NodeList
	if err := imports.Node(msg.UnmarshalVT); err != nil {
		panic(err.Error())
	}
	n.nl = &msg
	return n.nl
}