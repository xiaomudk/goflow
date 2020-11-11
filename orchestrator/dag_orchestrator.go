package orchestrator

import (
	"fmt"
	"goflow/dags"
	"goflow/logs"
	"time"

	"goflow/config"
	"goflow/k8sclient"

	"k8s.io/client-go/kubernetes"
)

// Orchestrator holds information for all DAGs
type Orchestrator struct {
	dagMap     map[string]*dags.DAG
	kubeClient kubernetes.Interface
	config     *config.GoFlowConfig
}

func newOrchestratorFromClientAndConfig(
	client kubernetes.Interface,
	config *config.GoFlowConfig,
) *Orchestrator {
	return &Orchestrator{make(map[string]*dags.DAG), client, config}
}

// NewOrchestrator creates an empty instance of Orchestrator
func NewOrchestrator(configPath string) *Orchestrator {
	return newOrchestratorFromClientAndConfig(
		k8sclient.CreateKubeClient(),
		config.CreateConfig(configPath),
	)
}

// AddDAG adds a DAG to the Orchestrator
func (orchestrator *Orchestrator) AddDAG(dag *dags.DAG) {
	logs.InfoLogger.Printf(
		"Added DAG %s which will run in namespace %s, with code %s",
		dag.Config.Name,
		dag.Config.Namespace,
		dag.Code,
	)
	orchestrator.dagMap[dag.Config.Name] = dag
}

// DeleteDAG removes a DAG from the orchestrator
func (orchestrator *Orchestrator) DeleteDAG(dagName string, namespace string) {
	dag := orchestrator.dagMap[dagName]
	dag.TerminateAndDeleteRuns()
	delete(orchestrator.dagMap, dagName)
}

// DAGs returns []DAGs with all DAGs present in the map
func (orchestrator Orchestrator) DAGs() []*dags.DAG {
	jobs := make([]*dags.DAG, 0, len(orchestrator.dagMap))
	for job := range orchestrator.dagMap {
		jobs = append(jobs, orchestrator.dagMap[job])
	}
	return jobs
}

// isDagPresent returns true if the given dag is present
func (orchestrator Orchestrator) isDagPresent(dag dags.DAG) bool {
	_, ok := orchestrator.dagMap[dag.Config.Name]
	return ok
}

// isStoredDagDifferent returns true if the given dag source code is different
func (orchestrator Orchestrator) isStoredDagDifferent(dag dags.DAG) bool {
	currentDag, _ := orchestrator.dagMap[dag.Config.Name]
	return currentDag.Code != dag.Code
}

// GetDag returns the DAG with the given name
func (orchestrator Orchestrator) GetDag(dagName string) *dags.DAG {
	dag, _ := orchestrator.dagMap[dagName]
	return dag
}

// CollectDAGs fills up the dag map with existing dags
func (orchestrator *Orchestrator) CollectDAGs() {
	dagSlice := dags.GetDAGSFromFolder(orchestrator.config.DAGPath)
	for _, dag := range dagSlice {
		fmt.Println(*dag)
		dagPresent := orchestrator.isDagPresent(*dag)
		if !dagPresent {
			orchestrator.AddDAG(dag)
			// stringJson, _ := json.MarshalIndent(orchestrator.dagMap, "", "\t")
			// fmt.Println(dag.Name, ":", string(stringJson))
		} else if dagPresent && orchestrator.isStoredDagDifferent(*dag) {
			logs.InfoLogger.Printf("Updating DAG %s which will run in namespace %s", dag.Config.Name, dag.Config.Namespace)
			logs.InfoLogger.Printf("Old DAG code: %s\n", orchestrator.GetDag(dag.Config.Name).Code)
			logs.InfoLogger.Printf("New DAG code: %s\n", dag.Code)
			// orchestrator.UpdateDag(&dag)
		}
	}
}

// Start begins the orchestrator event loop
func (orchestrator *Orchestrator) Start(cycleDuration time.Duration) {
	// serverRunning := true
	// for serverRunning {
	// 	orchestrator.CollectDAGs()
	// 	time.Sleep(cycleDuration)
	// }
	runs := 50
	for i := 0; i < runs; i++ {
		orchestrator.CollectDAGs()
		time.Sleep(cycleDuration)
	}
}
