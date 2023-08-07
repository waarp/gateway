==================
Modifier une règle
==================

.. program:: waarp-gateway rule update

Remplace les attributs de la règle donnée en paramètre par ceux fournis ci-dessous.
Les attributs omis resteront inchangés.

.. option:: <RULE>
   
   Le nom de la règle à modifier.

.. option:: <DIRECTION>

   Le sens de la règle à modifier. Peut être ``send`` ou ``receive``.

.. option:: -n <NAME>, --name=<NAME>

   Le nom de la règle de transfert. Doit être unique.

.. option:: -c <COMMENT>, --comment=<COMMENT>

   Un commentaire optionnel décrivant la règle.

.. option:: -d <DIRECTION>, --direction=<DIRECTION>

   Le sens de transfert des fichiers utilisant cette règle. Une règle
   peut être utilisée pour la réception (``receive``) ou l'envoi (``send``) de
   fichiers.

.. option:: -p <PATH>, --path=<PATH>

   Le chemin associé à la règle. Ce chemin sert à identifier la règle lors
   d'un transfert lorsque le protocole ne le permet pas. Par conséquent,
   ce chemin doit être unique.

.. option:: --local-dir=<DIRECTORY>

   Le chemin du dossier local des fichiers transférés. Dans le cas d'une règle
   d'envoi, ce dossier est utilisé comme source des fichiers. Dans le cas d'une
   règle de réception, ce dossier est utilisé comme destination des fichiers.
   Peut être un chemin relatif ou absolu. Le format du chemin dépend de l'OS de
   Waarp Gateway.

.. option:: --remote-dir=<DIRECTORY>

   Le chemin d'accès sur le serveur distant des fichiers transférés. Dans le cas
   d'une règle d'envoi, ce dossier est utilisé comme destination des fichiers.
   Dans le cas d'une règle de réception, ce dossier est utilisé comme source des
   fichiers. Ce chemin faisant partie d'un URI, il doit toujours être au format
   Unix standard.

.. option:: --tmp-dir=<DIRECTORY>

   Le chemin du dossier local temporaire des fichiers an cours de réception.
   Par conséquent, ce dossier n'est utile que pour les règles de réception.
   Le format du chemin dépend de l'OS de Waarp Gateway.

.. option:: -r <TASK>, --pre=<TASK>

   Un pré-traitement associé à la règle. Peut être répété plusieurs fois pour
   ajouter plusieurs traitements. Ces traitements seront exécutés avant chaque
   transfert dans l'ordre dans lequel ils ont été renseignés. Les traitements
   doivent être renseignés sous la forme d'un objet JSON avec 2 champs : le
   champ ``type`` et le champ ``args``. Le premier est une chaîne de caractères
   contenant la commande a exécuter, le second est un objet JSON contenant les
   arguments de la commande.

.. option:: -s <TASK>, --post=<TASK>

   Un post-traitement associé à la règle. Peut être répété plusieurs fois pour
   ajouter plusieurs traitements. Ces traitements seront exécutés après chaque
   transfert dans l'ordre dans lequel ils ont été renseignés. Les traitements
   doivent être renseignés sous la forme d'un objet JSON avec 2 champs : le
   champ ``type`` et le champ ``args``. Le premier est une chaîne de caractères
   contenant la commande a exécuter, le second est un objet JSON contenant les
   arguments de la commande.

.. option:: -e <TASK>, --err=<TASK>

   Un traitement d'erreur associé à la règle. Peut être répété plusieurs fois
   pour ajouter plusieurs traitements. Ces traitements seront exécutés en cas
   d'erreur dans l'ordre dans lequel ils ont été renseignés. Les traitements
   doivent être renseignés sous la forme d'un objet JSON avec 2 champs : le
   champ ``type`` et le champ ``args``. Le premier est une chaîne de caractères
   contenant la commande a exécuter, le second est un objet JSON contenant les
   arguments de la commande.

.. option:: -o <PATH>, --out_path=<PATH>

   .. deprecated:: 0.5.0

      Remplacé par les options ``--local-dir`` et ``--remote-dir``.

.. option:: -i <PATH>, --in_path=<PATH>

   .. deprecated:: 0.5.0
      
      Remplacé par les options ``--local-dir`` et ``--remote-dir``.

.. option:: -w <PATH>, --work_path=<PATH>

   .. deprecated:: 0.5.0

      Remplacé par ``--tmp-dir``.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' rule update règle_1 'send' -n 'règle_1_new' -c 'nouvelle règle de réception des fichiers avec SFTP' -p '/règle_1_new' -i '/règle_1_new/in' -o '/règle_1_new/out' --pre '{"type":"COPY","args":{"path":"chemin/copie"}}' --post '{"type":"DELETE","args":{}}' --err '{"type":"MOVE","args":{"path":"chemin/déplacement"}}'
