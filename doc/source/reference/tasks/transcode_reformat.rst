TRANSCODE&REFORMAT
==================

Le traitement ``TRANSCODE&REFORMAT`` combine les traitements :doc:`transcode`
et :doc:`reformat` : il change l'encodage textuel du fichier cible du transfert
et sa mise en forme en lignes ou en enregistrements, en une seule lecture du
fichier. Les arguments sont :

* ``fromCharset`` (*string*) - L'encodage actuel (source) du fichier.
* ``toCharset`` (*string*) - L'encodage souhaité (destination) du fichier.
* ``substitute`` (*bool*) - Écrit le caractère de substitution de l'encodage
  destination à la place d'un caractère qu'il ne sait pas représenter, au lieu
  d'échouer.
* ``records`` (*string*) - Le découpage du fichier source : ``none`` (par
  défaut) le fichier entier, ``lines`` des lignes, ``fixed`` des enregistrements
  de longueur fixe, ``rdw`` des enregistrements précédés de leur longueur (RDW),
  comme z/OS les écrit, ``articles`` des articles, aux longueurs que donnent
  les informations de transfert (voir plus bas).
* ``recordLength`` (*int*) - Avec ``fixed`` : la longueur des enregistrements,
  en octets, de 1 à 32760.
* ``trimPadding`` (*bool*) - Avec ``fixed`` : retire les espaces qui terminent
  chaque enregistrement.
* ``outputRecords`` (*string*) - La mise en forme du résultat, quand le fichier
  source est découpé : ``same`` (par défaut) comme le fichier source (des
  articles bout à bout), ``none`` sans rien entre les enregistrements,
  ``lines``, ``fixed`` ou ``rdw``.
* ``outputSeparator`` (*string*) - Avec ``lines`` : la fin de ligne écrite,
  ``LF`` (par défaut), ``CRLF`` ou ``NEL`` (fichier EBCDIC seulement).
* ``outputLength`` (*int*) - Avec ``fixed`` : la longueur des enregistrements
  du résultat, en octets, de 1 à 32760 ; un enregistrement plus court est
  complété par des espaces.
* ``truncate`` (*bool*) - Avec ``fixed`` ou ``rdw`` : coupe un enregistrement
  trop long au lieu d'échouer.

Les encodages se désignent comme pour :doc:`transcode`, qui en donne
:ref:`la liste <transcode-encodages>`. Les deux encodages doivent différer.

Sans ``records``, le fichier est converti en entier, comme le fait
``TRANSCODE``. Les informations de transfert qui décrivent les articles du
fichier restent en place, sauf leurs longueurs (``__articlesLengths__``) quand
un caractère peut changer de taille (de ou vers UTF-8, par exemple) : elles
ne décriraient plus le fichier et sont retirées, avec un avertissement dans le
journal. ``records: articles`` les garde justes.

Avec ``records``, seul le contenu des enregistrements est converti : un
séparateur ou un en-tête de longueur n'est jamais pris pour du texte. Une
combinaison d'arguments qui ne s'applique pas (``recordLength`` sans
``fixed``, ``outputSeparator`` sans ``lines``…) est refusée à l'enregistrement
de la règle.

Les fins de ligne sont des octets du fichier. Côté ASCII, ce sont LF et CR LF.
Côté EBCDIC, LF et CR LF sont aussi les octets ``0x0A`` et ``0x0D 0x0A`` : c'est
là que le module PeSIT de la passerelle découpe un fichier texte en articles,
quel que soit son encodage. Un fichier EBCDIC source a de plus ses lignes
reconnues sur ses propres fins de ligne, ``0x25`` et ``0x15`` ; ``NEL`` écrit
``0x15``. Avec ``same``, chaque ligne garde sa fin de ligne octet pour octet :
un fichier converti d'ASCII en EBCDIC par lignes garde ses ``0x0A``, et donc
ses articles quand il part en PeSIT.

.. code-block:: yaml

   pre:
     - type: TRANSCODE&REFORMAT
       args:
         fromCharset: "ISO 8859-1"
         toCharset: "IBM Code Page 297"
         records: "lines"

Informations de transfert
-------------------------

Les informations de transfert peuvent donner la longueur de chaque article du
fichier (``__articlesLengths__``), avec leur format (``__articlesFormat__``).
Une réception PeSIT les y écrit : le fichier reçu porte ses articles bout à
bout, sans rien entre eux. Une tâche ``SETINFO`` placée avant peut aussi les
poser, quel que soit le protocole du transfert (par exemple
``__articlesLengths__: "[16,26,5]"``). Avec ``records: articles``, le
traitement découpe le fichier à ces longueurs. Il
échoue, en laissant le fichier tel qu'il était, quand elles manquent ou quand
leur total ne fait pas la taille du fichier.

Après une conversion par enregistrements, le traitement réécrit ces clés pour
qu'elles décrivent le fichier converti, ce dont un envoi PeSIT qui suit a
besoin :

.. list-table::
   :header-rows: 1
   :widths: 35 65

   * - Mise en forme du résultat
     - Clés écrites
   * - ``lines`` avec ``LF`` ou ``CRLF``
     - ``__articlesSeparator__`` à la fin de ligne écrite, ``__articlesFormat__``
       à ``Variable`` ; ``__articlesLengths__`` retiré
   * - ``lines`` avec ``NEL``
     - ``__articlesSeparator__``, ``__articlesLengths__`` et ``__articlesFormat__``
       retirés : le module PeSIT ne découpe qu'à LF ou CR LF
   * - ``none``, ``fixed``, ou ``same`` depuis des articles
     - ``__articlesLengths__`` aux longueurs écrites ; ``__articlesFormat__`` à
       ``Variable`` pour ``none``, à ``Fixed`` pour ``fixed`` ; pour ``same``,
       inchangé, sauf ``Fixed`` qui passe à ``Variable`` quand les articles
       n'ont plus tous la même longueur ; ``__articlesSeparator__`` retiré
   * - ``rdw``
     - ``__articlesSeparator__``, ``__articlesLengths__`` et ``__articlesFormat__``
       retirés
   * - ``same`` depuis des lignes
     - Inchangées : chaque ligne garde sa fin de ligne. Seul
       ``__articlesLengths__`` est retiré, avec un avertissement dans le
       journal, quand la conversion peut changer la taille d'un caractère

Un enregistrement de plus de 65 535 octets ne tient pas dans un article : les
longueurs sont alors retirées, avec un avertissement dans le journal. Un
``SETINFO`` placé après le traitement l'emporte, puisqu'il s'exécute ensuite.

Ainsi, un fichier converti de lignes en EBCDIC avec ``outputRecords: none``
reste entièrement en EBCDIC, sans fin de ligne, et part en PeSIT découpé aux
longueurs de ses articles.
