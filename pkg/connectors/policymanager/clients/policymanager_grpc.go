// Copyright 2020 IBM Corp.
// SPDX-License-Identifier: Apache-2.0

package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"emperror.dev/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "fybrik.io/fybrik/pkg/connectors/protobuf"
	"fybrik.io/fybrik/pkg/model/policymanager"
	"fybrik.io/fybrik/pkg/model/taxonomy"
	random "fybrik.io/fybrik/pkg/random"
	"fybrik.io/fybrik/pkg/serde"
)

var _ PolicyManager = (*grpcPolicyManager)(nil)

const (
	intentKey     = "intent"
	roleKey       = "role"
	columnsKey    = "columns"
	redactKey     = "redact"
	columnNameKey = "column_name"
	nameKey       = "name"
	removeName    = "remove"
	encryptName   = "encrypt"
	allowAction   = "Allow"
	denyAction    = "Deny"
)

type grpcPolicyManager struct {
	pb.UnimplementedPolicyManagerServiceServer

	name       string
	connection *grpc.ClientConn
	client     pb.PolicyManagerServiceClient
}

// NewGrpcPolicyManager creates a PolicyManager facade that connects to a GRPC service
// You must call .Close() when you are done using the created instance
func NewGrpcPolicyManager(name, connectionURL string, connectionTimeout time.Duration) (PolicyManager, error) {
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()
	connection, err := grpc.DialContext(ctx, connectionURL, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("NewGrpcPolicyManager failed when connecting to %s", connectionURL))
	}
	return &grpcPolicyManager{
		name:       name,
		client:     pb.NewPolicyManagerServiceClient(connection),
		connection: connection,
	}, nil
}

func (m *grpcPolicyManager) GetPoliciesDecisions(
	in *policymanager.GetPolicyDecisionsRequest, creds string) (*policymanager.GetPolicyDecisionsResponse, error) {
	log.Println("open api request received for getting policy decisions: ", *in)
	appContext, _ := ConvertOpenAPIReqToGrpcReq(in, creds)
	log.Println("grpc application context to be used for getting policy decisions: ", appContext)

	result, err := m.client.GetPoliciesDecisions(context.Background(), appContext)
	if err != nil {
		log.Println("Error while obtaining get policies decisions: ", err)
		return nil, err
	}

	log.Println("GRPC result returned from GetPoliciesDecisions:", result)
	policyManagerResp, err := ConvertGrpcRespToOpenAPIResp(result)
	if err != nil {
		log.Println("Error during conversion to open api response: ", err)
		return nil, err
	}

	res, err := json.MarshalIndent(policyManagerResp, "", "\t")
	if err != nil {
		log.Println("Error during marshalling policy manager response: ", err)
		return nil, err
	}
	log.Println("Marshalled value of policy manager response: ", string(res))
	return policyManagerResp, nil
}

func (m *grpcPolicyManager) Close() error {
	return m.connection.Close()
}

func ConvertGrpcReqToOpenAPIReq(in *pb.ApplicationContext) (*policymanager.GetPolicyDecisionsRequest, string, error) {
	req := policymanager.GetPolicyDecisionsRequest{}
	action := policymanager.RequestAction{}
	resource := policymanager.Resource{}

	creds := in.GetCredentialPath()

	datasets := in.GetDatasets()
	// assume only one dataset is passed
	for i := 0; i < len(datasets); i++ {
		operation := datasets[i].GetOperation()
		destination := operation.GetDestination()
		action.Destination = destination
		operationType := operation.GetType()
		if operationType == pb.AccessOperation_READ {
			action.ActionType = taxonomy.ReadFlow
		}
		if operationType == pb.AccessOperation_WRITE {
			action.ActionType = taxonomy.WriteFlow
		}
		datasetID := datasets[i].GetDataset().GetDatasetId()
		resource.ID = taxonomy.AssetID(datasetID)
	}
	req.Resource = resource

	processingGeo := in.GetAppInfo().GetProcessingGeography()
	action.ProcessingLocation = taxonomy.ProcessingLocation(processingGeo)
	req.Action = action

	reqContext := make(map[string]interface{})
	properties := in.GetAppInfo().GetProperties()
	reqContext[intentKey] = properties[intentKey]
	reqContext[roleKey] = properties[roleKey]
	req.Context = taxonomy.PolicyManagerRequestContext{Properties: serde.Properties{Items: reqContext}}

	return &req, creds, nil
}

func ConvertOpenAPIReqToGrpcReq(in *policymanager.GetPolicyDecisionsRequest, creds string) (*pb.ApplicationContext, error) {
	credentialPath := creds
	action := in.Action
	processingGeo := action.ProcessingLocation
	log.Println("processingGeo: ", processingGeo)

	properties := make(map[string]string)
	reqContext := in.Context.Items
	if intent, ok := reqContext[intentKey].(string); ok {
		properties[intentKey] = intent
	}
	if role, ok := reqContext[roleKey].(string); ok {
		properties[roleKey] = role
	}

	appInfo := &pb.ApplicationDetails{ProcessingGeography: string(processingGeo), Properties: properties}

	datasetContextList := []*pb.DatasetContext{}
	resource := in.Resource
	datasetID := string(resource.ID)
	dataset := &pb.DatasetIdentifier{DatasetId: datasetID}

	destination := action.Destination
	actionType := action.ActionType

	var grpcActionType pb.AccessOperation_AccessType
	switch actionType {
	case taxonomy.ReadFlow:
		grpcActionType = pb.AccessOperation_READ
	case taxonomy.WriteFlow:
		grpcActionType = pb.AccessOperation_WRITE
	default: // default is read
		grpcActionType = pb.AccessOperation_READ
	}

	operation := &pb.AccessOperation{Type: grpcActionType, Destination: destination}
	datasetContext := &pb.DatasetContext{Dataset: dataset, Operation: operation}
	datasetContextList = append(datasetContextList, datasetContext)

	appContext := &pb.ApplicationContext{CredentialPath: credentialPath, AppInfo: appInfo, Datasets: datasetContextList}

	log.Println("Constructed GRPC appContext: ", appContext)

	return appContext, nil
}

//nolint:funlen,gocyclo
func ConvertOpenAPIRespToGrpcResp(
	out *policymanager.GetPolicyDecisionsResponse,
	datasetID string, op *pb.AccessOperation) (*pb.PoliciesDecisions, error) {
	res, err := json.MarshalIndent(out, "", "\t")
	log.Println("err :", err)
	log.Println("Marshalled response in ConvertOpenAPIRespToGrpcResp:", string(res))

	resultItems := out.Result
	enforcementActions := make([]*pb.EnforcementAction, 0)
	usedPolicies := make([]*pb.Policy, 0)

	for i := 0; i < len(resultItems); i++ {
		action := resultItems[i].Action
		log.Println("printing action ConvertOpenAPIRespToGrpcResp ", action)
		log.Println("printing action.AdditionalProperties ConvertOpenAPIRespToGrpcResp ", action.AdditionalProperties)
		name := string(action.Name)
		log.Println("name received in ConvertOpenAPIRespToGrpcResp", name)
		additionalProperties := action.AdditionalProperties.Items

		if strings.EqualFold(redactKey, name) {
			if additionalProperties != nil {
				fmt.Printf("type of additionalProperties\\[\"columns\"\\]: %s\n", reflect.TypeOf(additionalProperties[columnsKey]))
				if colNames, ok := additionalProperties[columnsKey].([]interface{}); ok {
					for j := 0; j < len(colNames); j++ {
						log.Println("colNames[j].(string)", colNames[j].(string))
						newEnforcementAction := &pb.EnforcementAction{Name: redactKey, Id: "redact-ID",
							Level: pb.EnforcementAction_COLUMN, Args: map[string]string{columnNameKey: colNames[j].(string)}}
						enforcementActions = append(enforcementActions, newEnforcementAction)

						policy := resultItems[i].Policy
						newUsedPolicy := &pb.Policy{Description: policy}
						usedPolicies = append(usedPolicies, newUsedPolicy)
					}
				} else {
					log.Println("additionalProperties does not have array of strings")
				}
			}
		}

		//nolint:dupl
		if strings.EqualFold(removeName, name) {
			if additionalProperties != nil {
				fmt.Printf("type of additionalProperties\\[\"columns\"\\]: %s\n", reflect.TypeOf(additionalProperties[columnsKey])) //nolint:revive
				if colNames, ok := additionalProperties[columnsKey].([]interface{}); ok {
					for j := 0; j < len(colNames); j++ {
						log.Println("colNames[j].(string)", colNames[j].(string)) //nolint:revive
						newEnforcementAction := &pb.EnforcementAction{Name: "removed", Id: "removed-ID",
							Level: pb.EnforcementAction_COLUMN, Args: map[string]string{columnNameKey: colNames[j].(string)}}
						enforcementActions = append(enforcementActions, newEnforcementAction)

						policy := resultItems[i].Policy
						newUsedPolicy := &pb.Policy{Description: policy}
						usedPolicies = append(usedPolicies, newUsedPolicy)
					}
				}
			}
		}

		//nolint:dupl
		if strings.EqualFold(encryptName, name) {
			if additionalProperties != nil {
				fmt.Printf("type of additionalProperties\\[\"columns\"\\]: %s\n", reflect.TypeOf(additionalProperties[columnsKey]))
				if colNames, ok := additionalProperties[columnsKey].([]interface{}); ok {
					for j := 0; j < len(colNames); j++ {
						log.Println("colNames[j].(string)", colNames[j].(string))
						newEnforcementAction := &pb.EnforcementAction{Name: "encrypted", Id: "encrypted-ID",
							Level: pb.EnforcementAction_COLUMN, Args: map[string]string{columnNameKey: colNames[j].(string)}}
						enforcementActions = append(enforcementActions, newEnforcementAction)

						policy := resultItems[i].Policy
						newUsedPolicy := &pb.Policy{Description: policy}
						usedPolicies = append(usedPolicies, newUsedPolicy)
					}
				}
			}
		}

		if strings.EqualFold("deny", name) {
			newEnforcementAction := &pb.EnforcementAction{Name: denyAction, Id: "Deny-ID", Level: pb.EnforcementAction_DATASET,
				Args: map[string]string{}}
			enforcementActions = append(enforcementActions, newEnforcementAction)

			policy := resultItems[i].Policy
			log.Println("policy got in ConvertOpenAPIRespToGrpcResp: ", policy)
			newUsedPolicy := &pb.Policy{Description: policy}
			usedPolicies = append(usedPolicies, newUsedPolicy)
		}

		if strings.EqualFold("allow", name) {
			newEnforcementAction := &pb.EnforcementAction{Name: allowAction, Id: "Allow-ID", Level: pb.EnforcementAction_DATASET,
				Args: map[string]string{}}
			enforcementActions = append(enforcementActions, newEnforcementAction)
		}
	}

	opaOperationDecision := &pb.OperationDecision{Operation: op, EnforcementActions: enforcementActions, UsedPolicies: usedPolicies}

	var datasetDecisionList []*pb.DatasetDecision
	var opaOperationDecisionList []*pb.OperationDecision
	opaOperationDecisionList = append(opaOperationDecisionList, opaOperationDecision)
	// Create a new *DatasetDecision
	dataset := &pb.DatasetIdentifier{DatasetId: datasetID}
	datasetDecison := &pb.DatasetDecision{Dataset: dataset, Decisions: opaOperationDecisionList}
	datasetDecisionList = append(datasetDecisionList, datasetDecison)

	policiesDecision := &pb.PoliciesDecisions{DatasetDecisions: datasetDecisionList}
	log.Println("returning policiesDecision in ConvertOpenAPIRespToGrpcResp: ", policiesDecision)
	return policiesDecision, nil
}

//nolint:funlen,gocyclo
func ConvertGrpcRespToOpenAPIResp(result *pb.PoliciesDecisions) (*policymanager.GetPolicyDecisionsResponse, error) {
	// convert GRPC response to Open Api Response - start
	// we dont get decision id returned from OPA from GRPC response. So we generate random hex string
	decisionID, _ := random.Hex(20) //nolint:revive,gomnd
	log.Println("decision id generated", decisionID)

	var datasetDecisions []*pb.DatasetDecision
	var decisions []*pb.OperationDecision
	datasetDecisions = result.GetDatasetDecisions()
	respResult := []policymanager.ResultItem{}

	// we assume only one dataset decision is passed
	for i := 0; i < len(datasetDecisions); i++ {
		datasetDecision := datasetDecisions[i]
		decisions = datasetDecision.GetDecisions()

		for j := 0; j < len(decisions); j++ {
			decision := decisions[j]
			var enfActionList []*pb.EnforcementAction
			var usedPoliciesList []*pb.Policy
			enfActionList = decision.GetEnforcementActions()
			usedPoliciesList = decision.GetUsedPolicies()

			for k := 0; k < len(enfActionList); k++ {
				enfAction := enfActionList[k]
				name := enfAction.GetName()
				level := enfAction.GetLevel()
				args := enfAction.GetArgs()
				log.Println("args received: ", args)
				log.Println("name received: ", name)
				log.Println("level received: ", level)
				policyManagerResult := policymanager.ResultItem{}

				if level == pb.EnforcementAction_COLUMN {
					actionOnCols := taxonomy.Action{}
					action := make(map[string]interface{})
					if name == redactKey {
						action[nameKey] = redactKey
						var colName string
						if _, ok := args[columnNameKey]; ok {
							colName = args[columnNameKey]
						} else {
							colName = args["column"]
						}
						action[columnsKey] = []string{colName}
					}
					if name == encryptName {
						action[nameKey] = encryptName
						action[columnsKey] = []string{args[columnNameKey]}
					}
					if name == removeName {
						action[nameKey] = removeName
						action[columnsKey] = []string{args[columnNameKey]}
					}

					actionBytes, errJSON := json.MarshalIndent(action, "", "\t")
					if errJSON != nil {
						return nil, fmt.Errorf("error Marshalling External Catalog Connector Response: %v", errJSON)
					}
					log.Println("actionBytes:", string(actionBytes))
					err := json.Unmarshal(actionBytes, &actionOnCols)
					if err != nil {
						return nil, fmt.Errorf("error in unmarshalling actionBytes : %v", err)
					}
					// just for printing
					actionOnColsBytes, errJSON := json.MarshalIndent(&actionOnCols, "", "\t")
					if errJSON != nil {
						return nil, fmt.Errorf("error Marshalling External Catalog Connector Response: %v", errJSON) //nolint:revive
					}
					log.Println("actionOnColsBytes: ", string(actionOnColsBytes))

					policyManagerResult.Action = actionOnCols
				}

				if level == pb.EnforcementAction_DATASET || level == pb.EnforcementAction_UNKNOWN {
					if name == denyAction {
						actionOnDataset := taxonomy.Action{}
						actionOnDataset.Name = denyAction
						policyManagerResult.Action = actionOnDataset
					}
				}
				if k < len(usedPoliciesList) {
					policy := usedPoliciesList[k].GetDescription()
					log.Println("usedPoliciesList[k].GetDescription()", policy)
					policyManagerResult.Policy = policy
				}
				if name != allowAction {
					// dont do anything For "Allow" action as this is convention now.
					// If we pass empty resultitem it means allow
					respResult = append(respResult, policyManagerResult)
				} else {
					log.Println("not doing any append to respResult for Allow action")
				}
			}
		}
	}
	// convert GRPC response to Open Api Response - end
	policyManagerResp := &policymanager.GetPolicyDecisionsResponse{DecisionID: decisionID, Result: respResult}

	log.Println("policyManagerResp in convGrpcRespToOpenApiResp", policyManagerResp)

	return policyManagerResp, nil
}
