# PeFi

- Household
    - sharing details
    - 1->N shared accounts
    - 1->N users
    - 1->N monthly budgets (1 per account)
- user
    - 1->N accounts
- account
    - account type
    - 1->N budget
- budget
    - date
    - details
    - 1->N balances
    - 1->N transactions
    - 1->N notes
- balance
    - account
    - budget
    - amount
- transaction
    - account
    - budget
    - amount


## New design
Accounts
 - AccountSnapshots (balance on date)
Transactions (used for prediction and spliting salary)
YieldModels (used for prediction)

Time machine allows you to run forward to a date and run prediction based on future transactions and yield models with
the given current balance snapshots.

### TODO

- Table with accounts as columns and snapshot dates as rows. Each cell is a snapshot of the balance of that account on that date.
- Prediction requires yield models for accounts
- TransactionTemplates will need to be modeled in a where they support the budget based transfers above
- Model startup shares
  - Model startup shares valuation based on company valuation and shares owned by the user
  - Model startup shares prediction based on future funding rounds and exit events

Can model the split using the transactionTemplates with fixed, percentage and remainder by doing the following:
- 1 salary account for each user
- 1 shared total account

- transaction to move % of salary to shared account first
- transactions to divide the remainder into personal accounts
- transactions to divide the shared total account into shared accounts
basically this means the users give a joint salary into the shared account which then splits normally.
the remaining in each salary account is divided into the subdivided accounts

## Views

- [ ] Index
  - [ ] [lowprio] CRUD list of users
  - [X] [high] CRUD list of accounts
  - [ ] [high] CRUD list of transactions
- [ ] Account
  - [X] CRUD list of snapshots
  - [ ] [high] CRUD list of yield models
- [ ] [high] Snapshot table
  - [ ] Table with accounts as columns and snapshot dates as rows. Each cell is a snapshot of the balance of that account on that date.
  - [ ] allow CRUD on cells rows
- [ ] [medium] PredictionView
