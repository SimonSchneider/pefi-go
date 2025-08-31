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

### TODO

- Handle darkmode for icons (move to templ and set stroke color based on daisy UI theme)
- A 100% transfer from a non fixed distribution will still 0 it out.
- Change the date for a snapshot row in the table view
- Add categories to accounts
- Support category grouping for graph
- Add categories to transfer-templates
- Add transfer run for a specific date for doing the money depositing split for that month
- Model startup shares
    - Model startup shares valuation based on company valuation and shares owned by the user
    - Model startup shares prediction based on future funding rounds and exit events
- Simulation Scenario combine different transaction templates into a sceanrio and see differences

## Modeling for transfering
Can model the split using the transactionTemplates with fixed, percentage and remainder by doing the following:

- 1 salary account for each user
- 1 shared total account

- transaction to move % of salary to shared account first
- transactions to divide the remainder into personal accounts
- transactions to divide the shared total account into shared accounts
  basically this means the users give a joint salary into the shared account which then splits normally.
  the remaining in each salary account is divided into the subdivided accounts

## New design

Accounts

- AccountSnapshots (balance on date)
  Transactions (used for prediction and spliting salary)
  InterestModels (used for prediction)

Time machine allows you to run forward to a date and run prediction based on future transactions and interest models
with
the given current balance snapshots.

## Tasks

### Core

- Tests for predictions

### Basics

- improve table view by allowing them to be configurable (which accounts to include)
- add categories and labels for accounts and transactions
- add charting for accounts and transactions with categories and labels as groups and filters
- make the prediction not run on request but rather create a prediction and save the result if it is slow

### Pages

#### Index

- list of transactions
- CRUD for transactions

#### Table

- color the cells based on if balance increased or decreased
- allow updates on cell rows
- allow updates on dates
- allow adding a new date row

#### Account

- CRUD for interest models
- list of interest models

#### Transaction

- CRUD for transaction

## Details

Asset class - (annualReturn, annualVolatility)

- Savings (0-2%, 0%)
- Funds (4%-8%, 6%-12%)
- Stocks (7%-10%, 15%-20%)
- Startup Stocks (20%-100%, 50%-200%)
- Private Company Stocks (5%-15%, 20%-40%)
- Real Estate (6%-10%, 10%-20%)

## Stuff

account = 300
transfer of max 80

loan limit is 0
loan = -200  -> transfer 80   min(80, (0-(-200))) -> min(80, 200)
loan = -70   -> transfer 70   min(80, (0-(-70)))  -> min(80, 70)

account limit is 100
account = 10 -> transfer 80   min(80, (100-10))   -> min(80, 90)
account = 30 -> transfer 70   min(80, (100-30))   -> min(80, 70)
