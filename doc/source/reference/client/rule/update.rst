==================
Modifier une règle
==================

.. program:: waarp-gateway rule update

.. describe:: waarp-gateway <ADDR> rule update <RULE>

Remplace les attributs de la règle donnée en paramètre par ceux fournis ci-dessous.
Les attributs omis resteront inchangés.

.. option:: -n <NAME>, --name=<NAME>

   Le nom de la règle de transfert. Doit être unique.

.. option:: -c <COMMENT>, --comment=<COMMENT>

   Un commentaire optionnel décrivant la règle.

.. option:: -p <PATH>, --path=<PATH>

   Le chemin associé à la règle. Ce chemin sert à identifier la règle lors
   d'un transfert lorsque le protocole ne le permet pas. Par conséquent,
   ce chemin doit être unique.

.. option:: -o <PATH>, --out_path=<PATH>

   Le chemin source des fichiers transférés. Lorsqu'un transfert est créé,
   le fichier sera cherché dans ce dossier. Ce chemin peut être laissé vide
   si l'on ne souhaite pas que la règle ait un dossier source spécifique.

.. option:: -i <PATH>, --in_path=<PATH>

   Le chemin de destination des fichiers transférés. Une fois un transfert
   terminé, le fichier est déposé dans ce dossier. Ce chemin peut être
   laissé vide si l'on ne souhaite pas que la règle ait un dossier destination
   spécifique.

.. option:: -r <TASK>, --pre=<TASK>

   Un pré-traitement associé à la règle. Peut être répété plusieurs fois
   pour ajouter plusieurs traitements. Ces traitements seront exécutés
   avant chaque transfert dans l'ordre dans lequel ils ont été renseignés.
   Les traitements doivent être renseignés sous la forme d'un objet JSON
   avec 2 champs: le champ `type` et le champ `args`. Le premier est une
   *string* contenant la commande a exécuter, le second est un objet JSON
   contenant les arguments de la commande.

.. option:: -s <TASK>, --post=<TASK>

   Un post-traitement associé à la règle. Peut être répété plusieurs fois
   pour ajouter plusieurs traitements. Ces traitements seront exécutés
   après chaque transfert dans l'ordre dans lequel ils ont été renseignés.
   Les traitements doivent être renseignés sous la forme d'un objet JSON
   avec 2 champs: le champ `type` et le champ `args`. Le premier est une
   *string* contenant la commande a exécuter, le second est un objet JSON
   contenant les arguments de la commande.

.. option:: -e <TASK>, --err=<TASK>

   Un traitement d'erreur associé à la règle. Peut être répété plusieurs
   fois pour ajouter plusieurs traitements. Ces traitements seront exécutés
   en cas d'erreur dans l'ordre dans lequel ils ont été renseignés.
   Les traitements doivent être renseignés sous la forme d'un objet JSON
   avec 2 champs: le champ `type` et le champ `args`. Le premier est une
   *string* contenant la commande a exécuter, le second est un objet JSON
   contenant les arguments de la commande.

|

**Exemple**

.. code-block:: bash

   waarp-gateway http://user:password@localhost:8080 rule add -n "règle_1_new" -c "nouvelle règle de réception des fichiers avec SFTP" -p "/règle_1_new" -i "/règle_1_new/in" -o "/règle_1_new/out" --pre='{"type":"COPY","args":{"path":"chemin/copie"}}' --post='{"type":"DELETE","args":{}}' --err='{"type":"MOVE","args":{"path":"chemin/déplacement"}}'
