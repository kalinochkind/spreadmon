package main

import (
	"errors"
	"strconv"
	"strings"

	"golang.org/x/net/html"
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

func getPageList(data string) (names []string, gids []string) {
	if data == "" {
		return nil, nil
	}
	doc, err := html.Parse(strings.NewReader(data))
	if err != nil {
		println(err.Error())
		return nil, nil
	}
	doc = doc.LastChild.LastChild.FirstChild
	if doc.FirstChild == doc.LastChild {
		doc = doc.NextSibling.FirstChild
		names = append(names, "")
		gids = append(gids, getAttr(doc, "id"))
		return
	}
	for i := doc.LastChild.FirstChild; i != nil; i = i.NextSibling {
		if i.Type == html.ElementNode && i.Data == "li" {
			val := getAttr(i, "id")
			if strings.HasPrefix(val, "sheet-button-") {
				gids = append(gids, strings.Split(val, "-")[2])
				names = append(names, i.FirstChild.FirstChild.Data)
			}
		}
	}
	return
}

func getPageListString(data string) *string {
	names, _ := getPageList(data)
	if names == nil {
		return nil
	}
	res := strings.Join(names, ", ")
	return &res
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
			return res
		}
	}
	t, _ := strconv.ParseInt(row, 10, 64)
	return int(t)
}

func calcHOffset(n *html.Node, col string) int {
	res := 0
	for th := n.FirstChild.FirstChild.FirstChild; th != nil; th = th.NextSibling {
		if getAttr(th, "style") == "" {
			continue
		}
		res++
		if th.FirstChild != nil && th.FirstChild.Data == col {
			return res
		}
	}
	return colToInt(col)
}

func extractCellValue(data string, gid string, row1 string, col1 string, row2 string, col2 string) (string, error) {
	defer func() {
		recover()
	}()
	doc, err := html.Parse(strings.NewReader(data))
	if err != nil {
		println(err.Error())
		return "", err
	}
	g := getGid(doc.LastChild.LastChild.FirstChild.NextSibling, gid)
	if g == nil {
		return "", errors.New("Page " + gid + " does not exist")
	}
	x1 := calcVOffset(g, row1)
	if x1 < 0 {
		return "", errors.New("Row " + row1 + " does not exist")
	}
	y1 := calcHOffset(g, col1)
	if y1 < 0 {
		return "", errors.New("Col " + col1 + " does not exist")
	}
	x2 := calcVOffset(g, row2)
	if x2 < 0 {
		return "", errors.New("Row " + row2 + " does not exist")
	}
	y2 := calcHOffset(g, col2)
	if y2 < 0 {
		return "", errors.New("Col " + col2 + " does not exist")
	}
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	cx, cy := 0, 0
	busy := make([][]bool, x2+1)
	for i := range busy {
		busy[i] = make([]bool, y2+1)
	}
	result := ""
	for tr := g.LastChild.FirstChild; tr != nil; tr = tr.NextSibling {
		if getAttr(tr, "style") == "" {
			continue
		}
		cx++
		if cx > x2 {
			break
		}
		cy = 0
		for td := tr.FirstChild; td != nil; td = td.NextSibling {
			if td.Data == "th" || getAttr(td, "class") == "freezebar-cell" {
				continue
			}
			cy++
			for cy <= y2 && busy[cx][cy] {
				cy++
			}
			if cy > y2 {
				break
			}
			if x1 <= cx && cx <= x2 && y1 <= cy && cy <= y2 {
				result += getText(td) + "\t"
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
				for ix := cx; ix < cx+int(dx) && ix <= x2; ix++ {
					for iy := cy; iy < cy+int(dy) && iy <= y2; iy++ {
						busy[ix][iy] = true
					}
				}
			}
		}
	}
	result = strings.Trim(result, "\t")
	return result, nil
}
