# README: Setup

This README provides step-by-step instructions to set up and start the assignment using Docker and Postman.

## API Details
The project includes the following API requests, which are pre-configured in the Postman collection:

1. Login Request
Authenticate users and retrieve an access token for further requests.

2. Check Account Balance
Fetch the current balance of a user’s account.

3. Submit Withdrawal
Submit a withdrawal request for a specified amount.

4. Approval Withdraw Claim
Approve a pending withdrawal request for processing.

## Cron Job for Failed Transactions

A cron job runs every 5 minutes to re-submit failed transactions to the chain. This ensures the system retries processing any transactions that previously encountered errors.

## Prerequisites

1. **Docker and Docker Compose**
    - Ensure Docker is installed and running on your system.
    - Install Docker Compose if it is not included with your Docker installation.

2. **Postman**
    - Install Postman to test the APIs using the provided `Application.postman_collection.json` file.

---

## Steps to Start the Application

### 1. Clone the Repository

If you haven’t already, clone the project repository to your local machine:
```bash
git clone <repository-url>
cd <repository-folder>
```

### 2. Start the Application with Docker Compose

Run the following command to build and start the application:
```bash
docker-compose up --build
```
This command will:
- Build the Docker images as defined in the `docker-compose.yml` file.
- Start the application and its dependencies (e.g., database, backend services).

### 3. Test the APIs with Postman

#### Import the Postman Collection

1. Open Postman.
2. Click on **File > Import** or the **Import** button.
3. Select the provided `Application.postman_collection.json` file.
4. The collection will be imported, and you will see a list of API endpoints grouped which includes:
   * Login Request
   * Check Account Balance
   * Submit Withdrawal
   * Approval Withdraw Claim - Staff 1
   * Approval Withdraw Claim - Manager 1
   * Approval Withdraw Claim - Manager 2
   * Approval Withdraw Claim - Manager 3

#### Configuration Environment Variables
1. Open the imported Postman collection.
2. Run the "Login API" request **four times** with appropriate credentials for:
   - `staff1`
   - `manager1`
   - `manager2`
   - `manager3`
3. Each time you run the "Login API" request, copy the returned token and set it in the Postman environment for the respective user (e.g., `staff1_token`, `manager1_token`, etc.).

   This step ensures you have the necessary tokens to execute the remaining requests in the collection seamlessly.

#### Edge Cases 
1. Login Api 
   * Could not connect to DB 
   * Could not find specific user based on given username and password
   * DB error
   * Error with generating token 
   * Error with updating token into DB
   * Token not updating in DB

2. Check Account Balance
   * Could not connect to DB 
   * Cannot find user based on token
   * No contact found on chain 
   * Error on retrieving account balance from chain
   
3. Submit Withdrawal
   * Could not connect to DB
   * Cannot find user based on token
   * Invalid Amount 
   * Could not connect to chain to check treasury balance
   * Treasury does not have enough balance
   * Claim request failed to insert into DB

4. Approval Withdraw Claim
   * Could not connect to db
   * Cannot find user based on token
   * User has no permission to approve withdrawal claims
   * Withdrawal Claim of ID : ? cannot be found 
   * Withdrawal Claim has already been approved by 2 managers and submitted to chain
   * Could not retrieve approval information from DB
   * Withdrawal Claim has been approved by you already
   * Claim request failed to insert into DB
   * No Claim request has been inserted into DB
   * Update withdrawal claim status failed
   * Claim amount parsing error
   * Cannot find specific User that made the claim request
   * Update withdrawal claim after submitting to chain failed