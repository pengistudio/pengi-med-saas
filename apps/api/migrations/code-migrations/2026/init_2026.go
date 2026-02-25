package y2026

import (
	"fmt"
	"pengi-med-saas/core/auth"
	"pengi-med-saas/core/database"
	company_models "pengi-med-saas/features/companies/models"
	permission_data "pengi-med-saas/features/permissions/data"
	permission_models "pengi-med-saas/features/permissions/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	user_models "pengi-med-saas/features/users/models"

	"gorm.io/gorm"
)

func init() {
	database.GlobalDBMap["DB20260214_1"] = database.DBExecute{
		ID: "DB20260214_1",
		Execute: func(db *gorm.DB) error {
			// 1. Create Tenant
			tenant := tenant_models.Tenant{
				Name: "Pengi Studio",
				Slug: "pengi-studio",
			}
			if err := db.Where(tenant_models.Tenant{Slug: tenant.Slug}).FirstOrCreate(&tenant).Error; err != nil {
				return fmt.Errorf("failed to create tenant: %w", err)
			}
			fmt.Printf("✅ Tenant '%s' created/found.\n", tenant.Name)

			// 2. Create Company
			company := company_models.Company{
				LegalName: "Pengi Med SaaS",
				TradeName: "Pengi",
				PlanCode:  "ENTERPRISE",
				TenantID:  tenant.ID,
			}
			if err := db.Where(company_models.Company{LegalName: company.LegalName, TenantID: tenant.ID}).FirstOrCreate(&company).Error; err != nil {
				return fmt.Errorf("failed to create company: %w", err)
			}
			fmt.Printf("✅ Company '%s' created/found.\n", company.TradeName)

			// 3. Create Roles
			roles := []string{"admin", "moderator", "user"}
			roleMap := make(map[string]user_models.Role)

			for _, roleName := range roles {
				role := user_models.Role{Role: roleName}
				if err := db.Where(user_models.Role{Role: roleName}).FirstOrCreate(&role).Error; err != nil {
					return fmt.Errorf("failed to create role '%s': %w", roleName, err)
				}
				roleMap[roleName] = role
				fmt.Printf("✅ Role '%s' created/found.\n", roleName)
			}

			// 4. Create User
			password, err := auth.HashPassword("password123")
			if err != nil {
				return fmt.Errorf("failed to hash password: %w", err)
			}

			user := user_models.User{
				UserName: "admin",
				Email:    "admin@pengi.com",
				Password: password, // Already hashed
			}

			// Check if user exists to avoid re-hashing or duplicate issues if FirstOrCreate tries to save
			var existingUser user_models.User
			if err := db.Where("user_name = ?", user.UserName).First(&existingUser).Error; err == nil {
				user = existingUser
				fmt.Printf("✅ User '%s' already exists.\n", user.UserName)
			} else if err == gorm.ErrRecordNotFound {
				if err := db.Create(&user).Error; err != nil {
					return fmt.Errorf("failed to create user: %w", err)
				}
				fmt.Printf("✅ User '%s' created.\n", user.UserName)
			} else {
				return fmt.Errorf("failed to query user: %w", err)
			}

			// 5. Create Environment (Link User, Company, Role)
			environment := user_models.Environment{
				UserID:    user.ID,
				CompanyID: company.ID,
				RoleID:    roleMap["admin"].ID,
				Name:      "Development",
			}

			if err := db.Where(user_models.Environment{UserID: user.ID, CompanyID: company.ID}).FirstOrCreate(&environment).Error; err != nil {
				return fmt.Errorf("failed to create environment: %w", err)
			}
			fmt.Printf("✅ Environment created/found for user '%s'.\n", user.UserName)

			// 6. Permissions (Optional: Add some permissions to roles if needed, for now just skipping as not explicitly requested with specific permissions)

			return nil
		},
	}

	database.GlobalDBMap["DB20260214_2"] = database.DBExecute{
		ID: "DB20260214_2",
		Execute: func(db *gorm.DB) error {
			// Retrieve the Admin Role to attach the new permissions
			var adminRole user_models.Role
			if err := db.Where(user_models.Role{Role: "admin"}).First(&adminRole).Error; err != nil {
				return fmt.Errorf("failed to find admin role: %w", err)
			}

			// 6. Permissions
			for _, perm := range permission_data.ClinicalPermissions {
				if err := db.Where(permission_models.Permission{BaseStringID: perm.BaseStringID}).FirstOrCreate(&perm).Error; err != nil {
					return fmt.Errorf("failed to create permission '%s': %w", perm.ID, err)
				}
				fmt.Printf("✅ Permission '%s' created/found.\n", perm.ID)

				// Assign permission to admin role
				if err := db.Model(&adminRole).Association("Permissions").Append(&perm); err != nil {
					return fmt.Errorf("failed to assign permission '%s' to admin role: %w", perm.ID, err)
				}
				fmt.Printf("✅ Assigned permission '%s' to admin role.\n", perm.ID)
			}
			return nil
		},
	}

	database.GlobalDBMap["DB20260214_3"] = database.DBExecute{
		ID: "DB20260214_3",
		Execute: func(db *gorm.DB) error {
			// 1. Retrieve first tenant and company
			var tenant1 tenant_models.Tenant
			if err := db.Where(tenant_models.Tenant{Slug: "pengi-studio"}).First(&tenant1).Error; err != nil {
				return fmt.Errorf("failed to find tenant 1: %w", err)
			}

			var company1 company_models.Company
			if err := db.Where(company_models.Company{LegalName: "Pengi Med SaaS"}).First(&company1).Error; err != nil {
				return fmt.Errorf("failed to find company 1: %w", err)
			}

			// Retrieve Admin Role
			var adminRole user_models.Role
			if err := db.Where(user_models.Role{Role: "admin"}).First(&adminRole).Error; err != nil {
				return fmt.Errorf("failed to find admin role: %w", err)
			}

			// 2. Create Tenant 2
			tenant2 := tenant_models.Tenant{
				Name: "Other Tenant",
				Slug: "other-tenant",
			}
			if err := db.Where(tenant_models.Tenant{Slug: tenant2.Slug}).FirstOrCreate(&tenant2).Error; err != nil {
				return fmt.Errorf("failed to create tenant 2: %w", err)
			}
			fmt.Printf("✅ Tenant '%s' created/found.\n", tenant2.Name)

			// 3. Create Company 2
			company2 := company_models.Company{
				LegalName: "Other Company Inc.",
				TradeName: "Other Company",
				PlanCode:  "PRO",
				TenantID:  tenant2.ID,
			}
			if err := db.Where(company_models.Company{LegalName: company2.LegalName, TenantID: tenant2.ID}).FirstOrCreate(&company2).Error; err != nil {
				return fmt.Errorf("failed to create company 2: %w", err)
			}
			fmt.Printf("✅ Company '%s' created/found.\n", company2.TradeName)

			password, err := auth.HashPassword("password123")
			if err != nil {
				return fmt.Errorf("failed to hash password: %w", err)
			}

			// 4. Create User 2 (Linked to Company 2)
			user2 := user_models.User{
				UserName: "admin2",
				Email:    "admin2@othercompany.com",
				Password: password,
			}
			var existingUser2 user_models.User
			if err := db.Where("user_name = ?", user2.UserName).First(&existingUser2).Error; err == nil {
				user2 = existingUser2
				fmt.Printf("✅ User '%s' already exists.\n", user2.UserName)
			} else if err == gorm.ErrRecordNotFound {
				if err := db.Create(&user2).Error; err != nil {
					return fmt.Errorf("failed to create user 2: %w", err)
				}
				fmt.Printf("✅ User '%s' created.\n", user2.UserName)
			} else {
				return fmt.Errorf("failed to query user 2: %w", err)
			}

			// Environment for User 2 -> Company 2
			env2 := user_models.Environment{
				UserID:    user2.ID,
				CompanyID: company2.ID,
				RoleID:    adminRole.ID,
				Name:      "Development",
			}
			if err := db.Where(user_models.Environment{UserID: user2.ID, CompanyID: company2.ID}).FirstOrCreate(&env2).Error; err != nil {
				return fmt.Errorf("failed to create environment 2: %w", err)
			}
			fmt.Printf("✅ Environment created/found for user '%s'.\n", user2.UserName)

			// 5. Create User 3 (Linked to Company 1)
			user3 := user_models.User{
				UserName: "admin3",
				Email:    "admin3@pengi.com",
				Password: password,
			}
			var existingUser3 user_models.User
			if err := db.Where("user_name = ?", user3.UserName).First(&existingUser3).Error; err == nil {
				user3 = existingUser3
				fmt.Printf("✅ User '%s' already exists.\n", user3.UserName)
			} else if err == gorm.ErrRecordNotFound {
				if err := db.Create(&user3).Error; err != nil {
					return fmt.Errorf("failed to create user 3: %w", err)
				}
				fmt.Printf("✅ User '%s' created.\n", user3.UserName)
			} else {
				return fmt.Errorf("failed to query user 3: %w", err)
			}

			// Environment for User 3 -> Company 1
			env3 := user_models.Environment{
				UserID:    user3.ID,
				CompanyID: company1.ID,
				RoleID:    adminRole.ID,
				Name:      "Development",
			}
			if err := db.Where(user_models.Environment{UserID: user3.ID, CompanyID: company1.ID}).FirstOrCreate(&env3).Error; err != nil {
				return fmt.Errorf("failed to create environment 3: %w", err)
			}
			fmt.Printf("✅ Environment created/found for user '%s'.\n", user3.UserName)

			return nil
		},
	}
}
