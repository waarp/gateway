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


Langue
======

Le sélecteur en haut à droite choisit le français ou l'anglais ; le choix est
conservé dans un cookie et vaut pour toute l'interface. Sans choix, la langue
est celle du navigateur.
