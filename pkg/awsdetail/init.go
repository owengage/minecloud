package awsdetail

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
)

func Init(detail *Detail) error {
	// TODO: Key pair
	// TODO: Security groups
	// TODO: Lambdas? Doesn't really fit into being able to do 'minecloud init'.
	// Maybe terraform or cloudformation is a better approach afterall. CDK?
	if err := InitServerRole(detail); err != nil {
		return err
	}

	return nil
}

func Deinit(detail *Detail) error {
	return DeinitServerRole(detail)
}

func InitServerRole(detail *Detail) error {
	iamServ := iam.New(detail.Session)
	detail.Logger.Info("creating role")
	roleOut, err := iamServ.CreateRole(&iam.CreateRoleInput{
		RoleName:    aws.String("Minecloud_ServerRole"),
		Description: aws.String("Allows access to required services for a Minecloud server to function."),

		AssumeRolePolicyDocument: aws.String(`{
            "Version": "2012-10-17",
            "Statement": [
                {
                    "Effect": "Allow",
                    "Action": "sts:AssumeRole",
                    "Principal": {"Service": "ec2.amazonaws.com"}
                }
            ]
        }`),
	})
	if err != nil {
		return err
	}

	detail.Logger.Info("creating instance profile")
	outInstProf, err := iamServ.CreateInstanceProfile(&iam.CreateInstanceProfileInput{
		InstanceProfileName: roleOut.Role.RoleName,
	})
	if err != nil {
		return err
	}

	detail.Logger.Info("adding role to instance profile")
	_, err = iamServ.AddRoleToInstanceProfile(&iam.AddRoleToInstanceProfileInput{
		RoleName:            roleOut.Role.RoleName,
		InstanceProfileName: outInstProf.InstanceProfile.InstanceProfileName,
	})
	if err != nil {
		return err
	}

	detail.Logger.Info("creating server policy")
	outPolicy, err := iamServ.CreatePolicy(&iam.CreatePolicyInput{
		PolicyName:  aws.String("Minecloud_ServerPolicy"),
		Description: aws.String("Allows a Minecloud server to access world/server files."),
		PolicyDocument: aws.String(`{
            "Version": "2012-10-17",
            "Statement": [
              {
                "Effect": "Allow",
                "Action": "s3:*",
                "Resource": [
                    "arn:aws:s3:::ogage-minecraft",
                    "arn:aws:s3:::ogage-minecraft/*"
                ]
              },
              {
                "Effect": "Allow",
                "Action": "ecr:GetAuthorizationToken",
                "Resource": "*"
              },
              {
                "Effect": "Allow",
                "Action": "ecr:*",
                "Resource": "arn:aws:ecr:eu-west-2:344791319371:repository/minecloud/server-wrapper"
              }
        	]
        }`),
	})
	if err != nil {
		return err
	}

	detail.Logger.Info("attaching policy to role")
	_, err = iamServ.AttachRolePolicy(&iam.AttachRolePolicyInput{
		PolicyArn: outPolicy.Policy.Arn,
		RoleName:  roleOut.Role.RoleName,
	})
	if err != nil {
		return err
	}

	return nil
}

func DeinitServerRole(detail *Detail) error {
	iamServ := iam.New(detail.Session)

	outPolicies, err := iamServ.ListPolicies(&iam.ListPoliciesInput{
		Scope: aws.String(iam.PolicyScopeTypeLocal),
	})
	if err != nil {
		return err
	}

	for _, policy := range outPolicies.Policies {
		if *policy.PolicyName == "Minecloud_ServerPolicy" {

			detail.Logger.Info("detaching server policy")
			_, err = iamServ.DetachRolePolicy(&iam.DetachRolePolicyInput{
				PolicyArn: policy.Arn,
				RoleName:  aws.String("Minecloud_ServerRole"),
			})
			if err != nil {
				detail.Logger.Warn("error:", err)
			}

			detail.Logger.Info("destroying server policy")
			_, err := iamServ.DeletePolicy(&iam.DeletePolicyInput{
				PolicyArn: policy.Arn,
			})
			if err != nil {
				detail.Logger.Warn("error:", err)
			}

			break
		}
	}

	detail.Logger.Info("detaching instance role")
	iamServ.RemoveRoleFromInstanceProfile(&iam.RemoveRoleFromInstanceProfileInput{
		InstanceProfileName: aws.String("Minecloud_ServerRole"),
		RoleName:            aws.String("Minecloud_ServerRole"),
	})

	detail.Logger.Info("destroying instance profile")
	iamServ.DeleteInstanceProfile(&iam.DeleteInstanceProfileInput{
		InstanceProfileName: aws.String("Minecloud_ServerRole"),
	})
	if err != nil {
		detail.Logger.Warn("error:", err)
	}

	detail.Logger.Info("destroying role")
	_, err = iamServ.DeleteRole(&iam.DeleteRoleInput{
		RoleName: aws.String("Minecloud_ServerRole"),
	})
	if err != nil {
		detail.Logger.Warn("error:", err)
	}

	return nil
}
