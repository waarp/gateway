#########################
Envoi d'un fichier en R66
#########################

.. _documentation Waarp R66: https://doc.waarp.org/waarp-r66/latest/fr/
.. _page de téléchargements: https://dl.waarp.org/

.. todo::
   A revoir

Nous allons maintenant mettre en place le second transfert : l'envoi d'un
fichier avec Gateway vers un serveur R66. Nous allons pour cela utiliser un
serveur Waarp R66 installé sur le serveur comme serveur de destination.

Pour pouvoir envoyer des fichiers, nous allons devoir ajouter un client R66
à la Gateway, y attacher un partenaire, puis créer un utilisateur, et une règle
de transfert.


Configuration du serveur R66 local
==================================

Pour notre exemple, nous supposerons qu'une installation de Waarp R66 basique
est présente et active sur la machine. Pour plus d'information sur la marche à
suivre pour installer Waarp R66, veuillez consulter la `documentation Waarp R66`_.
Dans notre exemple, l'instance de Waarp R66 s'appellera ``r66_server``.

La première étape est de récupérer le mot de passe de notre instance Waarp-R66.
Pour cela, ouvrons le fichier :file:`authent.xml`. Celui-ci devrait contenir une
entrée pour le serveur R66 lui-même, avec le mot de passe au niveau de la balise
``<key>``. Ce mot de passe étant crypté, nous allons utiliser
:program:`waarp-password` pour le décrypter. En se plaçant dans le dossier
racine de Waarp R66, et en fournissant le mot de passe écrit dans le fichier
après l'option ``-cpwd``, la commande est donc la suivante :

.. code-block:: shell-session

   $ ./bin/waarp-password.sh -ki "./etc/certs/cryptokey.des" -cpwd "04ebd92364b5aa60c29f04da512aca4be069dc6c6a842e25" -po "/dev/null" -clear
   ClearPwd: wm-clientpassword
   CryptedPwd: 04ebd92364b5aa60c29f04da512aca4be069dc6c6a842e25

Le mot de passe en clair devrait s'afficher dans la console (dans notre exemple
`wm-clientpassword`).

Si votre instance de Waarp R66 n'a aucune règle de transfert définie, il vous
faudra en créer une. Le plus simple est de copier le fichier :file:`rule.xml` se
trouvant dans les templates fournis avec Waarp R66 et de la coller dans le
dossier de configuration de l'instance R66.

Maintenant, il nous faut ajouter la *Gateway* dans la liste des partenaires de
l'agent Waarp R66. Pour cela, retournons dans le fichier :file:`authent.xml`.
Ajouter une entrée sous la forme suivante :

.. code-block:: xml

   <entry>
     <hostid>gw_r66user</hostid>
     <address>127.0.0.1</address>
     <port>6699</port>
     <isssl>false</isssl>
     <key>ff11b108880e33555a4b7c6a66065cbb622acefec8c3a302</key>
   </entry>

.. note::
   La ``<key>`` renseignée ici est la forme cryptée du mot de passe
   `gateway_password`. Si vous souhaitez utiliser un mot de passe différent, il
   vous faudra le chiffrer avec la commande suivante avant de l'ajouter à la
   configuration :

   .. code-block:: shell-session

      $ ./bin/waarp-password.sh -ki "./etc/certs/cryptokey.des" -pwd "$PASSWORD" -po "/dev/null"

Une fois le fichier sauvegardé, rechargez la configuration de Waarp R66 avec
la commande suivante :

.. code-block:: shell-session

   $ ./bin/waarp-r66client.sh r66_server loadconf

C'est suffisant, on va maintenant pouvoir se connecter au serveur ``r66_server``
sur le port ``6666``, avec l'utilisateur ``gw_r66user`` et le mot de passe
``gateway_password``.

Création d'un client R66
========================

Pour pouvoir envoyer des fichiers en R66 avec la Gateway, nous allons commencer
par ajouter un client R66 à la gateway :

.. code-block:: shell-session

   $ waarp-gateway client add --name "gw_r66_client" --protocol "r66"
   The client r66_client was successfully added.

Pour créer un client, nous devons, à minima, spécifier son nom et son protocole.
Optionnellement, il est également possible de configurer certains paramètres du
protocole via l'option ``--config``. Il est également possible d'attribuer une
adresse locale au client.

Une fois le client crée, nous devons le démarrer :

.. code-block:: shell-session

   $ waarp-gateway client start "gw_r66_client"
   The client r66_client was successfully started.

Création d'un partenaire R66
============================

Une fois le client créé, nous allons ajouter le partenaire R66 :

.. code-block:: shell-session

   $ waarp-gateway partner add --name "r66_server" --protocol "r66" --address "localhost:6666" --config "serverLogin:r66_server" --config "serverPassword:wm-clientpassword"
   The partner r66_server was successfully added.

Pour créer un partenaire, nous devons préciser son nom, le protocole de ce
serveur, ainsi que des informations additionnelles pour paramétrer le serveur
(ici l'adresse écoutée et le port).

.. seealso::

   Plus d'options de configuration sont disponibles pour les partenaires R66.

   Le détail des options est disponible :any:`ici <proto-config-r66>`

(Optionnel) Activation de TLS
-----------------------------

Optionnellement, si vous souhaitez sécuriser vos transfert vers ce partenaire
avec TLS, il faut altérer la configuration du partenaire en activant l'option
``isTLS`` ainsi :

.. code-block:: shell-session

   $ waarp-gateway partner update "r66_server" --config "serverLogin:waarp_r66" --config "serverPassword:sesame" --config "isTLS:true"

.. note::
   Il est nécessaire de re-entrer la configuration en entier pour que les
   valeurs de ``serverLogin`` et ``serverPassword`` ne soient pas perdues.

Attention, Gateway refuse les certificats TLS auto-signés. Si votre partenaire
R66 utilise un certificat auto-signé, il faudra l'ajouter à la liste des certificats
de confiance du partenaire comme ceci :

.. code-block:: shell-session

   $ waarp-gateway partner cert "r66_server" add --name "r66_server_cert" --certificate "cert.pem"
   The certificate r66_server was successfully added.

Il vous faudra également activer TLS dans la configuration de l'agent Waarp R66,
veuillez vous référer à la `documentation Waarp R66`_ pour la marche à suivre.


Création d'un utilisateur
-------------------------

Pour pouvoir se connecter au partenaire, nous devons maintenant créer un
utilisateur. Cela se fait en créant un "compte distant" dans la Gateway.
Cet utilisateur aura ``gw_r66user`` comme login et ``gateway_password`` comme
mot de passe (ceux définis plus tôt lors de la configuration de l'agent R66) :

.. code-block:: shell-session

   $ waarp-gateway account remote "r66_server" add --login "gw_r66user" --password "gateway_password"
   The account gw_r66user was successfully added.

L'utilisateur est maintenant créé. Pour pouvoir faire un transfert, nous devons
maintenant créer une :term:`règle` de transfert


Ajout d'un règle
----------------

Ici, nous voulons envoyer un fichier à la Gateway. La règle aura donc le sens
``send`` (« envoi ») : le sens des règles est toujours à prendre du point
de vu de la Gateway (si on envoi un fichier à Gateway, celle-ci le *reçoit*).
Attention, le nom de la règle doit être identique à celui de la règle définie
dans l'instance Waarp R66 (``default`` dans notre exemple).

Voici donc la commande pour créer la règle :

.. code-block:: shell-session

   $ waarp-gateway rule add --name "default" --direction "send"
   The rule default was successfully added.


Premier transfert
-----------------

Maintenant que nous avons un partenaire, un utilisateur et une règle, nous
pouvons effectuer un transfert. Créons d'abord un fichier à transférer, puis
envoyons-le avec Gateway :

.. code-block:: shell-session

   # echo "hello world!" > /var/lib/waarp-gateway/out/a-envoyer.txt

   $ transfer add --file "a-envoyer.txt" --way "send" --client "gw_r66_client" --partner "r66_server" --login "gw_r66user" --rule "default"
   The transfer of file a-envoyer.txt was successfully added.

Après avoir établi une connexion avec la Gateway, nous avons déposé un fichier
dans le dossier ``in`` de l'agent Waarp R66 avec la règle ``default``.

Nous pouvons vérifier que le transfert s'est bien passé dans l'historique des
transferts de la Gateway :

.. code-block:: shell-session

   $ waarp-gateway transfer list
   Transfers:
   [...]
   * Transfer 2 (as client) [DONE]
       Way:             send
       Protocol:        r66
       Rule:            default
       Client:          gw_r66_client
       Requester:       gw_r66user
       Requested:       r66_server
       Local filepath:  /etc/waarp-gateway/out/a-envoyer.txt
       Remote filepath: /a-envoyer.txt
       Start date:      2020-09-17T17:27:44Z
       End date:        2020-09-17T17:27:45Z

Le fichier disponible est maintenant dans le dossier ``in`` de Waarp R66.
Comme nous n'avons pas spécifié de dossier spécifique dans la règle, c'est le
dossier par défaut de l'instance qui est utilisé :

.. code-block:: shell-session

   $ ls -l ./data/r66_server/in
   total 4
   -rw-rw-r--. 1 waarp waarp 13 Sep 17 17:27 a-envoyer.txt


