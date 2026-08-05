.. _reference-tasks:

###########
Traitements
###########

Lors de l'ajout d'une règle, les traitements de la règle doivent être fournis
avec leurs arguments sous forme d'un objet JSON. Cet objet JSON contient 2
attributs:

* ``type`` (*string*) - Le type de traitement (voir liste ci-dessous).
* ``args`` (*object*) - Les arguments du traitement en format JSON. La structure
  de cet objet JSON dépend du type du traitement.

**Exemple**

.. code-block:: json

   {
     "type": "COPY",
     "args": {
       "path": "/backup"
     }
   }

.. _reference-tasks-list:

Liste des traitements
=====================

.. toctree::
   :maxdepth: 1

   copy
   copyrename
   delete
   remotedelete
   exec
   execmove
   execoutput
   move
   moverename
   rename
   transfer
   preregister
   sendmessage
   setinfo
   transcode
   change_newline
   archive
   extract
   icap
   email
   encrypt
   decrypt
   sign
   verify
   encrypt_sign
   decrypt_verify
   updateconf



.. _reference-tasks-substitutions:

Substitutions
=============

Les valeurs fournies dans l'objet ``args`` peuvent contenir des substitutions.

Les marqueurs de substitutions sont délimités par des signes dièse (``#``), et
sont valorisés au moment de l'exécution du traitement par les données
correspondant au transfert.

Les substitutions disponibles sont les suivantes :

============================ =============
Marqueur                     Signification
============================ =============
``#TRUEFULLPATH#``           Le chemin réel du fichier sur le disque
``#TRUEFILENAME#``           Le nom réel du fichier sur le disque
``#BASEFILENAME#``           Le nom du fichier sur disque (sans extension)
``#FILEEXTENSION#``          L'extension du fichier (avec le point inclus, ex: ``.txt``)
``#ORIGINALFULLPATH#``       Le chemin d'origine du fichier avant le transfert
``#ORIGINALFILENAME#``       Le nom d'origine du fichier avant le transfert
``#FILESIZE#``               La taille du fichier
``#HOMEPATH#``               Le dossier racine de la Gateway. Ce chemin est toujours
                             absolu.
``#INPATH#``                 Le dossier de réception. Ce chemin est toujours absolu.
``#OUTPATH#``                Le dossier d'envoi. Ce chemin est toujours absolu.
``#WORKPATH#``               Le dossier temporaire de réception. Ce chemin est toujours absolu.
``#RULE#``                   La règle utilisée par le transfert
``#DATE#``                   La date (au format ``AAAAMMJJ``) au moment de l'exécution
                             de la tâche
``#HOUR#``                   L'heure (au format ``HHMMSS``) au moment de l'exécution
                             de la tâche
``#TIMESTAMP(format)#``      Un timestamp du moment d'exécution des tâches. Le format
                             est constitué d'une suite de token qui seront remplacés
                             par leur valeur correspondante. La table de correspondance
                             peut être consultée :ref:`ci-dessous <ref-timestamp-format>`.
                             Par défaut, le format ``YYYY-MM-DD_HHmmss`` est utilisé.
``#STARTTIMESTAMP(format)#`` Un timestamp du début du transfert. Le format est
                             constitué d'une suite de token qui seront remplacés par
                             leur valeur correspondante. La table de correspondance
                             peut être consultée :ref:`ci-dessous <ref-timestamp-format>`.
                             Par défaut, le format ``YYYY-MM-DD_HHmmss`` est utilisé.
``#REMOTEHOST#``             L'identifiant du partenaire distant
``#LOCALHOST#``              L'identifiant du partenaire local
``#TRANSFERID#``             L'identifiant du transfert
``#REQUESTERHOST#``          L'identifiant du partenaire qui a demandé le transfert
``#REQUESTEDHOST#``          L'identifiant du partenaire qui a reçu la demande de
                             transfert
``#FULLTRANSFERID#``         Un identifiant "étendu" pour le transfert (de la forme
                             ``identifiantDeTransfert_idClient_idServeur``)
``#ERRORMSG#``               Message d'erreur (dans les traitements d'erreur)
``#ERRORCODE#``              Code d'erreur (dans les traitements d'erreur)
============================ =============

En plus de ces marqueurs standards, il est également possible de référencer les
:term:`infos de transfert` dans la définition d'un traitement. Pour ce faire,
le marqueur à utiliser est le suivant:

``#TI_<nom_de_clé>#`` où ``<nom_de_clé>`` est remplacée par le nom de la clé souhaitée.

À l'exécution, ce marqueur sera alors substitué par la valeur associée à la clé
renseignée.

Ces valeurs de substitutions sont également disponibles pour les programmes externes
appelés par les tâches EXEC sous forme de variables d'environnement. Ces variables
d'environnement ont exactement le même nom que leurs variables de substitution
correspondantes (ex: ``#TRUEFULLPATH#``). Par ailleurs, Waarp Gateway met à
disposition des programmes externes les variables d'environnement ``WAARP_CONFIG_FILE``
et ``WAARP_CONFIG_DIR`` contenant, respectivement, le chemin du fichier de
configuration de Waarp Gateway, et le dossier parent de ce fichier.

.. _ref-timestamp-format:

Formatage des *timestamps*
==========================

La table suivante indique les correspondance entre les différents tokens constituant
un format de *timestamp*, et leurs valeurs de remplacement une fois résolus.

À noter que tout caractère d'un format ne faisant pas partie d'un token sera
laissé inchangé. Par exemple, si un format de timestamp contient le caractère
*underscore* (``_``), celui-ci ne correspondant à aucun token dans la liste
ci-dessous, il sera donc laissé tel quel dans le timestamp final.

.. table::
   :widths: 30 10 20 60

   +------------------------+-------+------------------+--------------------------------------------+
   | Unité de temps         | Token | Valeur           | Description                                |
   +========================+=======+==================+============================================+
   | **Année**              | YYYY  | 2025             | Numéro d'année complet                     |
   |                        +-------+------------------+--------------------------------------------+
   |                        | YY    | 25               | Numéro d'année abrégé                      |
   +------------------------+-------+------------------+--------------------------------------------+
   | **Mois**               | MMMM  | January          | Nom complet du mois                        |
   |                        +-------+------------------+--------------------------------------------+
   |                        | MMM   | Jan              | Nom abrégé du mois                         |
   |                        +-------+------------------+--------------------------------------------+
   |                        | MM    | 01..12           | Numéro de mois (2 caractères)              |
   |                        +-------+------------------+--------------------------------------------+
   |                        | M     | 1..12            | Numéro de mois                             |
   +------------------------+-------+------------------+--------------------------------------------+
   | **Jour**               | DD    | 01..31           | Numéro du jour (2 caractères)              |
   |                        +-------+------------------+--------------------------------------------+
   |                        | D     | 1..31            | Numéro du jour                             |
   +------------------------+-------+------------------+--------------------------------------------+
   | **Jour de la semaine** | dddd  | Monday           | Nom complet du jour de la semaine          |
   |                        +-------+------------------+--------------------------------------------+
   |                        | ddd   | Mon              | Nom abrégé du jour de la semaine           |
   +------------------------+-------+------------------+--------------------------------------------+
   | **AM/PM**              | PM    | AM/PM            | Période du jour en majuscules              |
   |                        +-------+------------------+--------------------------------------------+
   |                        | pm    | am/pm            | Période du jour en minuscules              |
   +------------------------+-------+------------------+--------------------------------------------+
   | **Heure**              | HH    | 00..23           | Heure en format 24h (2 caractères)         |
   |                        +-------+------------------+--------------------------------------------+
   |                        | hh    | 01..12           | Heure en format 12h (2 caractères)         |
   |                        +-------+------------------+--------------------------------------------+
   |                        | h     | 1..12            | Heure en format 12h                        |
   +------------------------+-------+------------------+--------------------------------------------+
   | **Minutes**            | mm    | 00..59           | Minutes complétées (2 caractères)          |
   |                        +-------+------------------+--------------------------------------------+
   |                        | m     | 0..59            | Minutes                                    |
   +------------------------+-------+------------------+--------------------------------------------+
   | **Secondes**           | ss    | 00..59           | Secondes (2 caractères)                    |
   |                        +-------+------------------+--------------------------------------------+
   |                        | s     | 0..59            | Secondes                                   |
   +------------------------+-------+------------------+--------------------------------------------+
   | **Fuseau horaire**     | tz    | UTC, MST, CET... | Nom du fuseau horaire                      |
   |                        +-------+------------------+--------------------------------------------+
   |                        | zz    | -06:00 .. +06:00 | Décalage du fuseau horaire avec séparateur |
   |                        +-------+------------------+--------------------------------------------+
   |                        | z     | -0600 .. +0600   | Décalage du fuseau horaire                 |
   +------------------------+-------+------------------+--------------------------------------------+

