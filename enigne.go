package engine

import (
	"fmt"
	"sort"

	rbt "github.com/emirpasic/gods/trees/redblacktree"
)

type ExchangePair int

const (
	ADA_ETH  ExchangePair = iota // value: 1, type: ExchangePair
	BTC_ETH                      // value: 2, type: ExchangePair
	CVC_ETH                      // value: 3, type: ExchangePair
	DASH_ETH                     // value: 4, type: ExchangePair
	INVALID_EXCHANGE_PAIR
)

type Engine struct {
	pair ExchangePair
	book *rbt.Tree
}

// NewEngine is the Matching Engine constructor
func NewEngine(pair ExchangePair) Engine {
	return Engine{
		pair: pair,
		book: rbt.NewWithIntComparator(),
		// writeLog: NewWriteLog(exchange.String()),
	}
}

//addOrder adds an order that cannot be filled any further to the orderbook
// func (engine *Engine) addOrder(order *Order) *Order {
func addOrder(book *rbt.Tree, order *Order) {

	treeNode, ok := book.Get(order.price)
	if !ok {
		node := NewTreeNode()
		node.upsert(order)
		book.Put(order.price, node)

	} else {
		node := treeNode.(*TreeNode)
		node.upsert(order)
		book.Put(order.price, treeNode)
	}

}

//executeOrder walk the orderbook and match asks and bids that can fill
func executeOrder(book *rbt.Tree, executingOrder *Order) (*Order, []Match) {
	// myPrintln(">>executeOrder executingOrder price", executingOrder.price, executingOrder) //

	var matches []Match

	if executingOrder.operation == BID {

		// start left
		it := book.Iterator()

		// get the begin node then next
		for it.Begin(); it.Next(); {
			nodePrice, node := it.Key().(int), it.Value().(*TreeNode)
			myPrintln(">>executeOrder executingOrder price", executingOrder.price, executingOrder) //
			myPrintln(">>executeOrder BID book.Iterator() ", nodePrice, node)                      //

			//Check price
			if nodePrice <= executingOrder.price {
				// Have to append to top level variable but am dealing with scoped binding as well :=,
				// so it takes an extra line

				myPrintln(">>executeOrder BID book.Iterator() nodePrice <= executingOrder.price", nodePrice, executingOrder.price) //

				_, nodeMatches := matchNode(node, executingOrder)
				// myPrintln(">>executeOrder matchNode executingOrderResult.NumberOutstanding", executingOrderResult.NumberOutstanding, executingOrderResult)
				// myPrintln(">>executeOrder matchNode matches", nodeMatches)

				// myPrintln(">>executeOrder matchNode matches", nodeMatches)

				// ord = nodeOrderResult //??
				for _, nodeMatch := range nodeMatches {
					if nodeMatch.Number > 0 {
						matches = append(matches, nodeMatch)
					}
				}

			} else {
				//skip this node, too expensive (The cheapest ask could be higher than this bid)
				continue
			}

			if executingOrder.NumberOutstanding == 0 {
				// if we have 0 outstanding we can quit
				break
			}

		}
		return executingOrder, matches
	} else if executingOrder.operation == ASK {
		// fmt.Println("executeOrder ASK matchingOrder.Price", matchingOrder.price) //

		// start left?
		it := book.Iterator()

		// get the end element(highest) then previous
		for it.End(); it.Prev(); {
			nodePrice, node := it.Key().(int), it.Value().(*TreeNode)

			myPrintln(">>executeOrder executingOrder price", executingOrder.price, executingOrder) //
			myPrintln(">>executeOrder ASK book.Iterator() ", nodePrice, node)                      //

			//Check price to sell high?
			if nodePrice >= executingOrder.price {
				myPrintln(">>executeOrder ASK book.Iterator() nodePrice >= executingOrder.price", nodePrice, executingOrder.price) //

				// nodeOrderResult, nodeFills := matchNode(node, ord)
				_, nodeMatches := matchNode(node, executingOrder)
				// myPrintln(">>executeOrder matchNode executingOrderResult.NumberOutstanding", executingOrderResult.NumberOutstanding, executingOrderResult)
				myPrintln(">>executeOrder matchNode matches", nodeMatches)

				// ord = nodeOrderResult
				for _, fill := range nodeMatches {
					if fill.Number > 0 {
						matches = append(matches, fill)
					}
				}

			} else {
				myPrintln(">>executeOrder ASK book.Iterator() nodePrice < executingOrder.price", nodePrice, executingOrder.price) //
				//skip this node, too expensive (The cheapest ask could be higher than this bid)
				continue
			}

			if executingOrder.NumberOutstanding == 0 {
				// if we have 0 outstanding we can quit
				break
			}

		}
		return executingOrder, matches
	} else {
		// Not a valid bid/ask
	}

	return &Order{}, nil
}

// func (d *BookManager) Run(in <-chan Order, out chan<- Fill) {
func (engine *Engine) Run(order *Order) {
	// for order := range in {
	switch order.operation {
	case ASK:
		myPrintln("\n>*Run ASK for Higher price", order.price, order)

		//Ask things
		executedOrder, fills := executeOrder(engine.book, order)
		// myPrintln("Run ASK executedOrder", executedOrder)
		myPrintln(">Run ASK fills", fills)

		if executedOrder.NumberOutstanding > 0 {
			addOrder(engine.book, order)
		}

		// fmt.Println("ASK fill", fills)
		// //Write to WAL
		// d.writeLog.logFills(fills)
		// //Send fills to message bus
		// for _, fill := range fills {
		// 	out <- fill
		// }

		//
		// printOrderbook(engine.book) //

	case BID:
		myPrintln("\n*Run BID for Lower price", order.price, order)

		// Bid Operations
		executedOrder, matches := executeOrder(engine.book, order)
		// myPrintln("\nRun BID executedOrder", executedOrder)
		for _, fill := range matches {
			// if fill.Number > 0 {
			myPrintln("\nRun BID matches", fill)
			// }
		}

		if executedOrder.NumberOutstanding > 0 {
			addOrder(engine.book, order)
		}

		// fmt.Println("BID fill", fills)
		//Write to WAL
		// d.writeLog.logFills(fills)
		//Send fills to message bus
		// for _, fill := range fills {
		// 	out <- fill
		// }

		//
		// printOrderbook(engine.book) //
	case CANCEL:
		//Cancel an order
		// fill := cancelOrder(d.book, order.ID)

		// fmt.Println("CANCEL fill", fill)
		// d.writeLog.logFill(fill)
		// out <- fill

	default:
		//Drop the message
		fmt.Println("Invalid Order Type")
	}
	// }
	// printOrderbook(engine.book) //

}

//matchNode takes an order and fills it against a node, NOT IDEMPOTENT
func matchNode(node *TreeNode, matchingOrder *Order) (*Order, []Match) {
	// myPrintln("===>matchNode matchingOrder", ord.price, ord)

	//TODO
	//We only deal with ask and bid
	if matchingOrder.operation == CANCEL || matchingOrder.operation == INVALID_OPERATION {
		return matchingOrder, []Match{}
	}

	// fmt.Println("===>matchNode node.orders", node.orders)
	orders := node.sortedOrders()
	// fmt.Println("===>matchNode node.sortedOrders ", orders)

	activeOrder := matchingOrder //?
	var matches []Match

	for _, oldOrder := range orders {
		if activeOrder.operation != oldOrder.operation {
			// myPrintln("===>matchNode matchingOrder.operation != oldOrder.operation")
			// myPrintln("===>matchNode matchingOrder.NumberOutstanding", matchingOrder.NumberOutstanding, matchingOrder)
			// myPrintln("===>matchNode oldOrder.NumberOutstanding     ", oldOrder.NumberOutstanding, oldOrder)

			// If the current order can fill new order
			if oldOrder.NumberOutstanding >= matchingOrder.NumberOutstanding {
				myPrintln("===>oldOrder.NumberOutstanding >= matchingOrder.NumberOutstanding", oldOrder.NumberOutstanding, matchingOrder.NumberOutstanding)
				partialFill := []*Order{activeOrder, oldOrder}
				closed := []*Order{activeOrder}

				if oldOrder.NumberOutstanding-matchingOrder.NumberOutstanding == 0 { //??
					// myPrintln("===>oldOrder.NumberOutstanding-matchingOrder.NumberOutstanding == 0")
					closed = append(closed, oldOrder)
					node.delete(oldOrder.id)
					nodeMatch := NewMatch(activeOrder.pair, activeOrder.NumberOutstanding, oldOrder.price, partialFill, closed)

					//Order is filled
					activeOrder.NumberOutstanding = 0
					matches = append(matches, nodeMatch)

				} else { // Update old order
					myPrintln("===>oldOrder.NumberOutstanding-matchingOrder.NumberOutstanding != 0")

					oldRemaining := oldOrder.NumberOutstanding - activeOrder.NumberOutstanding
					oldOrder.NumberOutstanding = oldRemaining

					nodeMatch := NewMatch(activeOrder.pair, activeOrder.NumberOutstanding, oldOrder.price, partialFill, closed)

					//Order is matched
					activeOrder.NumberOutstanding = 0
					matches = append(matches, nodeMatch)

					node.upsert(oldOrder)
				}

			} else { // If the current order is too small to fill the new order
				myPrintln("===>oldOrder.NumberOutstanding < matchingOrder.NumberOutstanding", oldOrder.NumberOutstanding, matchingOrder.NumberOutstanding)
				// //How do we delete the old order?
				node.delete(oldOrder.id)

				partialFill := []*Order{activeOrder, oldOrder}
				closed := []*Order{oldOrder}
				nodeMatch := NewMatch(activeOrder.pair, oldOrder.NumberOutstanding, oldOrder.price, partialFill, closed)

				activeOrder.NumberOutstanding = activeOrder.NumberOutstanding - oldOrder.NumberOutstanding
				matches = append(matches, nodeMatch)

			}

		}
	}

	return activeOrder, matches
}

func (n TreeNode) sortedOrders() []*Order {
	orders := make([]*Order, 0)
	for _, v := range n.orders {
		orders = append(orders, v)
	}

	sort.Slice(orders[:], func(i, j int) bool {
		return orders[i].Timestamp < orders[j].Timestamp
	})

	return orders
}

// func (n *TreeNode) delete(id uuid.UUID) {
func (n *TreeNode) delete(id uint64) {
	delete(n.orders, id)
	if len(n.orders) == 0 {
		myPrintln("len(n.orders)==0", n)
	}
	//if(n.orders)
}
