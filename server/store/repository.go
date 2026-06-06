package store

import "go.woodpecker-ci.org/woodpecker/v3/server/model"

type UserRepository interface {
	Get(id int64) (*model.User, error)
	GetByRemoteID(forgeID int64, remoteID model.ForgeRemoteID) (*model.User, error)
	GetByLogin(forgeID int64, login string) (*model.User, error)
	List(p *model.ListOptions) ([]*model.User, error)
	Create(user *model.User) error
	Update(user *model.User) error
	Delete(user *model.User) error
}

type RepoRepository interface {
	Get(id int64) (*model.Repo, error)
	GetByForgeID(forgeID int64, remoteID model.ForgeRemoteID) (*model.Repo, error)
	GetByName(fullName string) (*model.Repo, error)
	List(user *model.User, owned, active bool, filter *model.RepoFilter) ([]*model.Repo, error)
	ListLatest(user *model.User) ([]*model.Feed, error)
	Create(repo *model.Repo) error
	Update(repo *model.Repo) error
	Delete(repo *model.Repo) error
}

type PipelineRepository interface {
	Get(id int64) (*model.Pipeline, error)
	GetByNumber(repo *model.Repo, number int64) (*model.Pipeline, error)
	List(repo *model.Repo, p *model.ListOptions, filter *model.PipelineFilter) ([]*model.Pipeline, error)
	LatestByRepoIDs(repoIDs []int64) ([]*model.Pipeline, error)
	Create(pipeline *model.Pipeline, steps ...*model.Step) error
	Update(pipeline *model.Pipeline) error
	Delete(pipeline *model.Pipeline) error
}

type OrgRepository interface {
	Get(id int64) (*model.Org, error)
	FindByName(name string, forgeID int64) (*model.Org, error)
	Create(org *model.Org) error
	Update(org *model.Org) error
}

func NewUserRepository(s Store) UserRepository {
	return userRepository{store: s}
}

func NewRepoRepository(s Store) RepoRepository {
	return repoRepository{store: s}
}

func NewPipelineRepository(s Store) PipelineRepository {
	return pipelineRepository{store: s}
}

func NewOrgRepository(s Store) OrgRepository {
	return orgRepository{store: s}
}

type userRepository struct {
	store Store
}

func (r userRepository) Get(id int64) (*model.User, error) {
	return r.store.GetUser(id)
}

func (r userRepository) GetByRemoteID(forgeID int64, remoteID model.ForgeRemoteID) (*model.User, error) {
	return r.store.GetUserByRemoteID(forgeID, remoteID)
}

func (r userRepository) GetByLogin(forgeID int64, login string) (*model.User, error) {
	return r.store.GetUserByLogin(forgeID, login)
}

func (r userRepository) List(p *model.ListOptions) ([]*model.User, error) {
	return r.store.GetUserList(p)
}

func (r userRepository) Create(user *model.User) error {
	return r.store.CreateUser(user)
}

func (r userRepository) Update(user *model.User) error {
	return r.store.UpdateUser(user)
}

func (r userRepository) Delete(user *model.User) error {
	return r.store.DeleteUser(user)
}

type repoRepository struct {
	store Store
}

func (r repoRepository) Get(id int64) (*model.Repo, error) {
	return r.store.GetRepo(id)
}

func (r repoRepository) GetByForgeID(forgeID int64, remoteID model.ForgeRemoteID) (*model.Repo, error) {
	return r.store.GetRepoForgeID(forgeID, remoteID)
}

func (r repoRepository) GetByName(fullName string) (*model.Repo, error) {
	return r.store.GetRepoName(fullName)
}

func (r repoRepository) List(user *model.User, owned, active bool, filter *model.RepoFilter) ([]*model.Repo, error) {
	return r.store.RepoList(user, owned, active, filter)
}

func (r repoRepository) ListLatest(user *model.User) ([]*model.Feed, error) {
	return r.store.RepoListLatest(user)
}

func (r repoRepository) Create(repo *model.Repo) error {
	return r.store.CreateRepo(repo)
}

func (r repoRepository) Update(repo *model.Repo) error {
	return r.store.UpdateRepo(repo)
}

func (r repoRepository) Delete(repo *model.Repo) error {
	return r.store.DeleteRepo(repo)
}

type pipelineRepository struct {
	store Store
}

func (r pipelineRepository) Get(id int64) (*model.Pipeline, error) {
	return r.store.GetPipeline(id)
}

func (r pipelineRepository) GetByNumber(repo *model.Repo, number int64) (*model.Pipeline, error) {
	return r.store.GetPipelineNumber(repo, number)
}

func (r pipelineRepository) List(repo *model.Repo, p *model.ListOptions, filter *model.PipelineFilter) ([]*model.Pipeline, error) {
	return r.store.GetPipelineList(repo, p, filter)
}

func (r pipelineRepository) LatestByRepoIDs(repoIDs []int64) ([]*model.Pipeline, error) {
	return r.store.GetRepoLatestPipelines(repoIDs)
}

func (r pipelineRepository) Create(pipeline *model.Pipeline, steps ...*model.Step) error {
	return r.store.CreatePipeline(pipeline, steps...)
}

func (r pipelineRepository) Update(pipeline *model.Pipeline) error {
	return r.store.UpdatePipeline(pipeline)
}

func (r pipelineRepository) Delete(pipeline *model.Pipeline) error {
	return r.store.DeletePipeline(pipeline)
}

type orgRepository struct {
	store Store
}

func (r orgRepository) Get(id int64) (*model.Org, error) {
	return r.store.OrgGet(id)
}

func (r orgRepository) FindByName(name string, forgeID int64) (*model.Org, error) {
	return r.store.OrgFindByName(name, forgeID)
}

func (r orgRepository) Create(org *model.Org) error {
	return r.store.OrgCreate(org)
}

func (r orgRepository) Update(org *model.Org) error {
	return r.store.OrgUpdate(org)
}
