EXECMOVE
========

Le traitement ``EXECMOVE`` exécute un programme externe. Ce programme *doit*
déplacer le fichier au cours de son exécution, et *doit* également afficher le
nouveau chemin du fichier sur la sortie standard. Les arguments sont:

* ``path`` (*string*) - Le chemin du programme à exécuter.
* ``args`` (*string*) - Les arguments du programme.
* ``delay`` (*number*) - La durée d'exécution maximum du programme (en ms). Si
  le programme n'a pas terminé après que cette durée soit écoulée, le programme
  sera interrompu et le traitement sera considéré comme ayant échoué.

.. note::
   La valeur de sortie du programme détermine si l'exécution a réussie ou échoué.
   - une valeur de 0 signifie un succès
   - une valeur de 1 signifie un succès avec message d'avertissement
   - toute autre valeur est considérée comme un échec
