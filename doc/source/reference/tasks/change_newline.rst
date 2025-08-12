CHNEWLINE
=========

Le traitement ``CHNEWLINE`` permet de changer le ou les caractères utilisés pour
marquer la fin de ligne dans le fichier de transfert (en supposant que le
fichier contienne du texte). Les arguments sont:

* ``from`` (*string*) - La séquence de fin de ligne actuelle du fichier.
* ``to`` (*string*) - La séquence de fin de ligne souhaitée.

.. note::
   Les caractères utilisés pour marquer les fins de ligne étant généralement
   non-imprimables, ceux-ci doivent donc être échappés afin de pouvoir être
   représentés sous forme d'une *string*. Il existe plusieurs manières d'échapper
   ces caractères. Prenons par exemple la séquence ``CR LF``, communément utilisée
   sous Windows. Les caractères qui la compose peuvent être représentés des
   manières suivantes :

   - En utilisant directement leurs séquences d'échappement Go. Pour notre
     exemple, cela donnerait ``\r\n``.
   - En renseignant leurs valeurs hexadécimales, ce qui donnerait ``\x0D\x0A``.
   - En renseignant leurs valeurs octales, ce qui donnerait ``\015\012``.
   - En renseignant leurs valeurs Unicode, ce qui donnerait ``\u000D\u000A``.