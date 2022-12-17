/*BSD 2-Clause License*/
/*Copyright (c) 2014-2019, Alexander Willing*/
/*All rights reserved.*/

/*Redistribution and use in source and binary forms, with or without*/
/*modification, are permitted provided that the following conditions are met:*/

/*1. Redistributions of source code must retain the above copyright notice, this*/
   /*list of conditions and the following disclaimer.*/

/*2. Redistributions in binary form must reproduce the above copyright notice,*/
   /*this list of conditions and the following disclaimer in the documentation*/
   /*and/or other materials provided with the distribution.*/

/*THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"*/
/*AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE*/
/*IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE*/
/*DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE*/
/*FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL*/
/*DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR*/
/*SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER*/
/*CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,*/
/*OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE*/
/*OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.*/
package game

// from engine/rendertext.cpp
const (
	green   = "\f0" // player talk
	blue    = "\f1" // "echo" command
	yellow  = "\f2" // gameplay messages
	red     = "\f3" // important errors
	gray    = "\f4"
	magenta = "\f5"
	orange  = "\f6"
	white   = "\f7"

	save    = "\fs"
	restore = "\fr"
)

func wrap(s, color string) string {
	return save + color + s + restore
}

func Green(s string) string   { return wrap(s, green) }
func Blue(s string) string    { return wrap(s, blue) }
func Yellow(s string) string  { return wrap(s, yellow) }
func Red(s string) string     { return wrap(s, red) }
func Gray(s string) string    { return wrap(s, gray) }
func Magenta(s string) string { return wrap(s, magenta) }
func Orange(s string) string  { return wrap(s, orange) }
func White(s string) string   { return wrap(s, white) }

func Success(s string) string { return Green(s) }
func Fail(s string) string    { return Orange(s) }
func Error(s string) string   { return Red(s) }
