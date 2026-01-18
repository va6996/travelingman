package orm

import (
	"time"

	"github.com/va6996/travelingman/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

type User struct {
	ID           uint `gorm:"primaryKey"`
	Email        string
	PasswordHash string
	FullName     string
	CreatedAt    time.Time
	// Relationships
	TravelGroups []TravelGroup `gorm:"many2many:user_groups;"`
}

func (u *User) ToPB() *pb.User {
	if u == nil {
		return nil
	}
	return &pb.User{
		Id:           int64(u.ID),
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		FullName:     u.FullName,
		CreatedAt:    timestamppb.New(u.CreatedAt),
		// Passports/Licenses not yet in ORM
	}
}

func UserFromPB(p *pb.User) *User {
	if p == nil {
		return nil
	}
	return &User{
		ID:           uint(p.Id),
		Email:        p.Email,
		PasswordHash: p.PasswordHash,
		FullName:     p.FullName,
		CreatedAt:    p.CreatedAt.AsTime(),
	}
}

func CreateUser(db *gorm.DB, pbUser *pb.User) error {
	user := UserFromPB(pbUser)
	if err := db.Create(user).Error; err != nil {
		return err
	}
	// Write back ID
	pbUser.Id = int64(user.ID)
	return nil
}

func GetUser(db *gorm.DB, id uint) (*pb.User, error) {
	var user User
	err := db.Preload("TravelGroups").First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return user.ToPB(), nil
}

func UpdateUser(db *gorm.DB, pbUser *pb.User) error {
	user := UserFromPB(pbUser)
	return db.Save(user).Error
}
