package main

import (
	"strings"
	"strconv"
	"golang.org/x/net/html"
	"errors"
)

func colToInt(col string) int {
	col = strings.ToUpper(col)
	ans := 0
	for _, c := range col {
		ans *= 26
		ans += int(c) - 'A' + 1
	}
	return ans
}

func getAttr(node *html.Node, arg string) string {
	for _, v := range node.Attr {
		if v.Key == arg {
			return v.Val
		}
	}
	return ""
}

func getGid(n *html.Node, gid string) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if getAttr(c, "id") == gid {
			return c.FirstChild.FirstChild
		}
	}
	return nil
}

func getText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	res := ""
	for i := n.FirstChild; i != nil; i = i.NextSibling {
		res += getText(i)
	}
	return res
}

func calcVOffset(n *html.Node, row string) int {
	res := 0
	for tr := n.LastChild.FirstChild; tr != nil; tr = tr.NextSibling {
		if getAttr(tr, "style") == "" {
			continue
		}
		res++
		if tr.FirstChild.FirstChild.FirstChild.Data == row {
			return res;
		}
	}
	return -1
}

func calcHOffset(n *html.Node, col string) int {
	res := 0
	for th := n.FirstChild.FirstChild.FirstChild.NextSibling; th != nil; th = th.NextSibling {
		if getAttr(th, "style") == "" {
			continue
		}
		res++
		if th.FirstChild.Data == col {
			return res
		}
	}
	return -1
}

func extractCellValue(data string, gid string, row string, col string) (string, error) {
	doc, err := html.Parse(strings.NewReader(data))
	if err != nil {
		println(err.Error())
		return "", err
	}
	g := getGid(doc.LastChild.LastChild.FirstChild.NextSibling, gid)
	if g == nil {
		return "", errors.New("Page " + gid + " does not exist")
	}
	x := calcVOffset(g, row)
	if x < 0 {
		return "", errors.New("Row " + row + " does not exist")
	}
	y := calcHOffset(g, col)
	if y < 0 {
		return "", errors.New("Col " + col + " does not exist")
	}
	cx, cy := 0, 0
	busy := make([][]bool, x+1)
	for i := range busy {
		busy[i] = make([]bool, y+1)
	}
	for tr := g.LastChild.FirstChild; tr != nil; tr = tr.NextSibling {
		if getAttr(tr, "style") == "" {
			continue
		}
		cx++
		if cx > x {
			break
		}
		cy = 0
		for td := tr.FirstChild; td != nil; td = td.NextSibling {
			if td.Data == "th" || getAttr(td, "class") == "freezebar-cell" {
				continue
			}
			cy++
			for cy <= y && busy[cx][cy] {
				cy++
			}
			if cy > y {
				break
			}
			if cx == x && cy == y {
				return getText(td), nil
			}
			colspan, rowspan := getAttr(td, "colspan"), getAttr(td, "rowspan")
			if colspan != "" || rowspan != "" {
				if colspan == "" {
					colspan = "1"
				}
				if rowspan == "" {
					rowspan = "1"
				}
				dx, _ := strconv.ParseInt(rowspan, 10, 64)
				dy, _ := strconv.ParseInt(colspan, 10, 64)
				for ix := cx; ix < cx+int(dx) && ix <= x; ix++ {
					for iy := cy; iy < cy+int(dy) && iy <= y; iy++ {
						busy[ix][iy] = true
					}
				}
			}
		}
	}
	return "", nil
}
