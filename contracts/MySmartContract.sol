// SPDX-License-Identifier: GPL-3.0

pragma solidity ^0.8.0;
/**
* @title Storage
* @dev store or retrieve variable value
*/

contract Storage {
    mapping(address => uint256) private _balances;
    uint256 private _totalSupply;
    address public treasury;
    string public name = "TreasuryToken";
    string public symbol = "TT";
    uint8 public decimals = 18;
    event Withdraw(address indexed from, address indexed to, uint256 value);
    event Mint( uint256 value);

    constructor(address _treasury) {
        require(_treasury != address(0), "Treasury address cannot be zero");
        treasury = _treasury;
        _totalSupply = 100000;
        _balances[treasury] = 100000;
    }

    modifier onlyTreasury() {
        require(msg.sender == treasury, "Only treasury can mint");
        _;
    }

    function mint(uint256 amount) public onlyTreasury {
        _totalSupply += amount;
        _balances[treasury] += amount;
        emit Mint(amount);
    }

    function balanceOf(address account) public view returns (uint256) {
        return _balances[account];
    }

    function withdraw(address recipient, uint256 amount) public returns (bool) {
        require(_balances[treasury] >= amount, "Insufficient balance");
        _balances[treasury] -= amount;
        _balances[recipient] += amount;
        emit Withdraw(treasury, recipient, amount);
        return true;
    }
}