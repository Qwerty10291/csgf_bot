package utils

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/Knetic/govaluate"
)

type tokenType int

func init() {
	rand.Seed(time.Now().UnixMicro())
}

const (
	tokenNum tokenType = iota
	tokenSign tokenType = iota
	tokenExpr tokenType = iota
)
const signs = "+-*"

func MathematicExpressionGenerator() (string, int) {
	expr := mathExprRecursion(strconv.Itoa(1 + rand.Intn(10)), tokenNum, 0, 1, 3)
	answerExpr, err := govaluate.NewEvaluableExpression(expr)
	if err != nil{
		panic(err)
	}
	answer, err := answerExpr.Eval(nil)
	if err != nil{
		panic(err)
	}
	return expr, int(answer.(float64))
}

func mathExprRecursion(expr string, lastToken tokenType, depth int, maxDepth int, maxNumsCount int) string {
	count := 1
	for {
		switch lastToken {
		case tokenNum, tokenExpr:
			expr += string(signs[rand.Intn(3)])
			lastToken = tokenSign
			break
		case tokenSign:
			if chance(0.1) && depth <= maxDepth{
				expr += fmt.Sprintf("(%s)", mathExprRecursion(strconv.Itoa(1 + rand.Intn(10)), tokenNum, depth + 1, maxDepth, maxNumsCount))
				lastToken = tokenExpr
			} else {
				expr += strconv.Itoa(rand.Intn(10))
				count += 1
				lastToken = tokenNum
				if chance(0.3) || count >= maxNumsCount{
					return expr
				}
				break
			}	
		}
	}
}

func chance(chance float32) bool{
	return rand.Float32() <= chance
}