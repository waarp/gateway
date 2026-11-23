.. _user-webui-login:

#########
Connexion
#########

Adresse ``/webui/login``, la première page de l'interface : toute adresse de
l'interface y mène tant qu'aucune session n'est ouverte, et la connexion
ramène ensuite à la page demandée.

La page demande un identifiant et un mot de passe : ceux d'un
:ref:`utilisateur <user-webui-users>` de la Gateway, les mêmes que pour l'API
REST et la ligne de commande. Le bouton à droite du mot de passe l'affiche en
clair le temps de le vérifier. Un mot de passe oublié se change par un
administrateur, dans :doc:`utilisateurs` : la page ne propose pas de le
réinitialiser.


Refus
=====

Un identifiant ou un mot de passe incorrect est dit sous le bouton, sans
préciser lequel des deux ; l'identifiant saisi reste en place. Après 5 échecs
pour un même compte depuis une même adresse, ou 25 échecs depuis une même
adresse tous comptes confondus, les tentatives sont refusées pendant 1 minute,
avec le même message, et cette durée double à chaque nouveau verrouillage
(section ``[security]`` du :ref:`fichier de configuration
<configuration-file>`).


Session
=======

La session reste ouverte tant que l'utilisateur agit ; elle expire après 20
minutes sans action et, dans tous les cas, après 24 heures. Un compte n'a
qu'une session à la fois : se connecter depuis un autre navigateur met fin à
la précédente. Quand la session a pris fin, la page de connexion le dit et la
connexion ramène à la page demandée. *Se déconnecter*, dans le menu du compte
en haut à droite, ferme la session ; la page de connexion le dit aussi.
L'adresse ``/webui/logout`` ouverte directement, depuis un favori ou un lien,
ne ferme pas la session : elle mène à l'accueil.


.. _user-webui-login-proxy:

Origine des requêtes et mandataire inverse
==========================================

L'interface n'accepte une action qui modifie l'état (connexion, création,
modification, suppression, reprise d'un transfert…) que si elle vient de
l'interface elle-même. Le navigateur dit d'où vient chaque requête ; une
requête envoyée par un autre site, par une application servie sur un autre
port du même hôte ou par un autre hôte du même domaine est refusée avec le
code ``403``. Les pages elles-mêmes s'ouvrent depuis n'importe où, et l'API
REST n'est pas concernée.

Derrière un mandataire inverse (*reverse proxy*), une règle est à respecter :
**le mandataire transmet l'en-tête** ``Host`` **tel qu'il l'a reçu du
navigateur**, port compris. La Gateway compare l'hôte que le navigateur
annonce comme origine à celui auquel la requête est adressée ; si le
mandataire remplace ``Host`` par le nom interne de la Gateway, ou en retire le
port, les deux diffèrent et toute action est refusée, connexion comprise. Le
cas se présente quand l'interface est jointe en HTTP par un nom d'hôte ; en
HTTPS, le navigateur indique lui-même que la requête vient de l'interface.

Un refus se reconnaît à deux signes :

- la page affiche que la requête a été refusée parce qu'elle ne vient pas de
  l'interface, et nomme l'en-tête ``Host`` ;
- le journal de la Gateway porte un avertissement qui donne l'origine et
  l'hôte reçus :

  .. code-block:: text

     2026/11/23 10:54:44.216919 [WARNING ] admin: Refused the request POST "/webui/login", which comes from another origin: Origin "http://gateway.example.com", Host "10.0.0.5:8080", Sec-Fetch-Site "". Behind a reverse proxy, the Host header must reach the gateway as the browser sent it.

Dans cet exemple, le navigateur a joint ``gateway.example.com`` et le
mandataire a transmis ``10.0.0.5:8080`` : il faut régler le mandataire pour
qu'il conserve l'en-tête ``Host`` d'origine.


Langue
======

Le sélecteur en haut à droite choisit le français ou l'anglais ; le choix est
conservé dans un cookie et vaut pour toute l'interface. Sans choix, la langue
est celle du navigateur.
